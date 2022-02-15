package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
	writeToConsole := flag.Bool("o", false, fmt.Sprintf("output, if true will write to console"))
	flag.Parse()

	monitor, err := createMonitorFromFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	monitor.writeToConsole = *writeToConsole

	fmt.Println("begin monitoring")

	// create an error channel for each process being monitored
	pErrChan := make(chan error, len(monitor.Processes))
	var wg sync.WaitGroup
	for _, k := range monitor.Processes {
		wg.Add(1)
		//fmt.Println("monitoring process:", k)
		// this goroutine will run indefinitely
		go monitor.checkProcess(k, pErrChan, &wg)
	}

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

func (monitor *Monitor) checkProcess(processName string, errChan chan error, wg *sync.WaitGroup) {
	fmt.Println("monitoring for process", processName)

	counter := 0
	// buffers to compare against changes in processes
	var b1 bytes.Buffer
	var b2 string

	//repCmd := ""

	for {

		cmd := exec.Command("pgrep", "-l", processName)

		// create a buffer for reads and writes
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
		// fmt.Println("b1", b1.String())
		// fmt.Println("b2", b2)
		// fmt.Println("comparing", strings.Compare(b1.String(), b2))
		if strings.Compare(b1.String(), b2) != 0 {
			log.Fatal(processName, " changed status")
		}
		b2 = b1.String()

		time.Sleep(time.Second * 1)
		counter++
		//fmt.Println("Have monitored", processName, " for ", counter, "seconds")
		b1.Reset()
	}

	// check the output
	//fmt.Println(&b2)

	defer wg.Done()
}
