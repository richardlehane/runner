package main

import (
	"bytes"
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
	auth, url := os.Getenv("RUNNER_AUTH"), os.Getenv("RUNNER_JOBS")
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
	var jobs runner.Jobs
	if err := json.Unmarshal(body, &jobs); err != nil {
		log.Fatalf("Error unmarshalling, got: %v", err)
	}
	lgs := make(map[string][]runner.Report)
	urls := make(map[string]string)
	for _, j := range jobs {
		// execute each job and if it has a log key, then store its output for reporting
		st, dur, out, err := execute(j)
		if j.LogKey != "" {
			lgs[j.LogKey] = append(lgs[j.LogKey], runner.Report{
				Detail:   j.Detail,
				Start:    st,
				Duration: dur,
				Output:   out.String(),
				Err:      err,
			})
			if j.URL != "" {
				urls[j.LogKey] = j.URL
			}
		}
	}
	for k, v := range lgs {
		if err := post(auth, urls[k], runner.Log{Detail: k, Reports: v}); err != nil {
			log.Print(err)
		}
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
		cmd := exec.Command(j.Cmd[0], args...)
		cmd.Stdout = ioutil.Discard
		cmd.Stderr = ioutil.Discard
		if err := cmd.Run(); err != nil {
			stderr = "Error on first run: " + err.Error()
			return
		}
	}
	cmd := exec.Command(j.Cmd[0], args...)
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