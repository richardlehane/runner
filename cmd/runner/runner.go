package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/richardlehane/runner"
)

func main() {
	var (
		jobs    runner.Jobs
		lg      []runner.Report
		lgkey   string
		posturl string
		batch   = time.Now()
	)
	auth, url, mach := os.Getenv("RUNNER_AUTH"), os.Getenv("RUNNER_URL"), os.Getenv("RUNNER_MACH")
	if auth == "" || url == "" {
		log.Fatal("Must set RUNNER_AUTH and RUNNER_URL environment variables")
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Could not connect to runner jobs at %s, got: %v", url, err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body, got: %v", err)
	}
	resp.Body.Close()
	if err := json.Unmarshal(body, &jobs); err != nil {
		log.Fatalf("Error unmarshalling, got: %v", err)
	}
	for _, j := range jobs {
		// if log key is different, progressive post last jobs
		if lgkey == "" {
			lgkey = j.LogKey
		} else if j.LogKey != "" && j.LogKey != lgkey {
			if err := post(auth, posturl, runner.Log{Label: lgkey, Batch: batch, Machine: mach, Reports: lg}); err != nil {
				log.Print(err)
			}
			lg = nil
			lgkey = j.LogKey
		}
		// execute each job and if it has a log key, then store its output for reporting
		st, dur, out, err := execute(j)
		if j.LogKey != "" {
			lg = append(lg, runner.Report{
				Detail:   j.Detail,
				Start:    st,
				Duration: dur,
				Err:      err,
			})
			if out != nil {
				if j.Base64 {
					lg[len(lg)-1].Output = base64.StdEncoding.EncodeToString(out.Bytes())
				} else {
					lg[len(lg)-1].Output = out.String()
				}
			}
			if j.URL != "" {
				posturl = j.URL
			}
		}
	}
	if lg == nil {
		return
	}
	if err := post(auth, posturl, runner.Log{Label: lgkey, Batch: batch, Machine: mach, Reports: lg}); err != nil {
		log.Print(err)
	}
}

func post(auth, url string, content interface{}) error {
	body, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth("runner", auth)
	resp, err := client.Do(req)
	byt, _ := ioutil.ReadAll(resp.Body)
	log.Print(string(byt))
	resp.Body.Close()
	return err
}

func execute(j runner.Job) (start time.Time, duration time.Duration, stdout *bytes.Buffer, stderr string) {
	var args []string
	if len(j.Cmd) == 0 {
		return
	}
	if len(j.Cmd) > 1 {
		args = j.Cmd[1:]
	}
	var output io.Writer
	buf, ebuf := &bytes.Buffer{}, &bytes.Buffer{}
	if j.Save != "" {
		f, err := os.Create(j.Save)
		if err != nil {
			stderr = err.Error()
			return
		}
		defer f.Close()
		if j.SendOut {
			output = io.MultiWriter(f, buf)
		} else {
			output = f
		}
	} else if j.SendOut {
		output = buf
	} else {
		output = ioutil.Discard
	}
	if j.RunTwice {
		var cmd *exec.Cmd
		if j.Timeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
			defer cancel()
			cmd = exec.CommandContext(ctx, j.Cmd[0], args...)
		} else {
			cmd = exec.Command(j.Cmd[0], args...)
		}
		cmd.Stdout = ioutil.Discard
		cmd.Stderr = ioutil.Discard
		if err := cmd.Run(); err != nil {
			stderr = "Error on first run: " + err.Error()
			return
		}
	}
	if j.Background != nil && len(j.Background.Cmd) > 0 {
		var bargs []string
		if len(j.Background.Cmd) > 1 {
			bargs = j.Background.Cmd[1:]
		}
		bcmd := exec.Command(j.Background.Cmd[0], bargs...)
		if err := bcmd.Start(); err != nil {
			stderr = "Error starting background process: " + err.Error()
			return
		}
		defer bcmd.Wait()
		<-time.After(j.Background.Delay)
	}
	var cmd *exec.Cmd
	if j.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, j.Cmd[0], args...)
	} else {
		cmd = exec.Command(j.Cmd[0], args...)
	}
	cmd.Stdout = output
	cmd.Stderr = ebuf
	start = time.Now()
	runerr := cmd.Run()
	duration = time.Since(start)
	if runerr != nil {
		stderr = runerr.Error() + "\n"
	}
	stderr += ebuf.String()
	if j.SendOut {
		stdout = buf
	}
	return
}
