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
	"strings"
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
			go monitor.notifyErr(pErr, server, monitor.Config.Recipients)
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

	counter := 0
	var b1 bytes.Buffer
	var b2 string

	for {
		cmd := exec.Command("pgrep", "-l", processName)
		// use the byte buffer for writing process list
		cmd.Stdout = &b1

		cmd.Start()
		cmd.Wait()

		if counter == 0 {
			b2 = b1.String()
			counter++
			b1.Reset()
			continue
		}
		// compare previous output to current output
		// react to changes in status
		if strings.Compare(b1.String(), b2) != 0 {
			errChan <- fmt.Sprintf("something changed with %s", processName)
			time.Sleep(time.Second * 30)
		}
		b2 = b1.String()

		// check every second
		time.Sleep(time.Second * monitor.Config.CheckFrequency)
		counter++
		// reset the buffer
		b1.Reset()
	}
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
