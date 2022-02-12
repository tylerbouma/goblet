package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	writeToConsole := flag.Bool("o", false, fmt.Sprintf("output, if true will write to console"))
	flag.Parse()

	monitor, err := createMonitorFromFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	monitor.writeToConsole = *writeToConsole

	fmt.Println("monitoring")

	// create an error channel for each process being monitored
	pErrChan := make(chan error, len(monitor.Processes))
	var wg sync.WaitGroup
	for _, k := range monitor.Processes {
		wg.Add(1)
		fmt.Println("monitoring process:", k)
		monitor.checkProcess(k, pErrChan, &wg)
	}

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
	fmt.Println("checking for process", processName)

	cmd1 := exec.Command("ps", "aux")
	cmd2 := exec.Command("grep", processName)

	// connect the ps and grep commands
	r, w := io.Pipe()
	cmd1.Stdout = w
	cmd2.Stdin = r

	// create a buffer for reads and writes
	var b2 bytes.Buffer
	cmd2.Stdout = &b2

	cmd1.Start()
	cmd2.Start()
	cmd1.Wait()
	w.Close()
	cmd2.Wait()

	// check the output
	fmt.Println(&b2)

	defer wg.Done()
}
