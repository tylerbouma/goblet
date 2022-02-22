package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Monitor struct {
	// Will parse a yaml file and create a monitor struct based on that data
	Processes []string
	Config    struct {
		Recipients     string        `yaml:"recipients"`
		CheckFrequency time.Duration `yaml:"checkFreq"`
	}
	writeToConsole bool
}

func main() {

	defaultConfigFile := "./go-monitor.yml"
	configFile := flag.String("c", defaultConfigFile, fmt.Sprintf("config file path, default = %s", defaultConfigFile))
	writeToConsole := flag.Bool("o", false, "output, if true will write to console")
	flag.Parse()

	monitor, err := createMonitorFromFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	monitor.writeToConsole = *writeToConsole

	server, err := monitor.serverInfo()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("begin monitoring on server:", server)

	// create an error channel for each process being monitored
	pErrChan := make(chan string, len(monitor.Processes))
	var wg sync.WaitGroup
	for _, k := range monitor.Processes {
		wg.Add(1)
		// this goroutine will run indefinitely
		go monitor.monitorProcess(k, pErrChan, &wg)
	}

	// read the results from the error channel
	go func() {
		for {
			pErr := <-pErrChan
			// go monitor.notifyErr(pErr, server, monitor.Config.Recipients)
			fmt.Println(pErr)
		}
	}()

	wg.Wait()

}

func createMonitorFromFile(configFile string) (monitor *Monitor, error error) {

	conf, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(error)
	}

	fmt.Println("using:", configFile)

	error = yaml.Unmarshal(conf, &monitor)
	if err != nil {
		log.Fatal(err)
	}

	// validate the yaml file was built correctly
	error = monitor.validate()

	return
}

func (monitor *Monitor) validate() error {

	if len(monitor.Processes) < 1 {
		return errors.New("need at least one process to monitor")
	} else {
		fmt.Printf("Processes %s\n", monitor.Processes)
	}

	if monitor.Config.Recipients == "" {
		return errors.New("recipient list is empty")
	}

	return nil
}

func (monitor *Monitor) serverInfo() (server string, err error) {
	server, err = os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (monitor *Monitor) monitorProcess(processName string, errChan chan string, wg *sync.WaitGroup) {
	fmt.Println("monitoring for process", processName)

	var b1 bytes.Buffer
	waitFlag := true

	for {
		cmd := exec.Command("pgrep", "-l", processName)
		// use the byte buffer for writing process list
		cmd.Stdout = &b1

		cmd.Start()
		cmd.Wait()

		lines, err := countLines(&b1)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(0)
		}

		if lines != 0 && waitFlag {
			// we are making a first pass
			// ensuring the process actually exists
			waitFlag = false
		} else if lines == 0 && !waitFlag {
			fmt.Printf("Error: no process %s found running\n", processName)
			errChan <- processName
		}

	}
	wg.Done()
	// keep the channel open
	errChan <- ""
}

func countLines(r io.Reader) (int, error) {
	b := make([]byte, 32*1024)
	count := 0
	newLine := []byte{'\n'}

	for {
		l, err := r.Read(b)
		count += bytes.Count(b[:l], newLine)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func (monitor *Monitor) notifyErr(proc, server, recipient string) {
	// send a notification to the appropriate party
	// sender data
	from := "<email un>"
	password := "<email pw>"

	to := []string{
		recipient,
	}

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	message := []byte(fmt.Sprintf("test message %s", proc))

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("email sent successfully")

}
