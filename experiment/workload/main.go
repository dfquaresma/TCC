package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

var (
	expId       = flag.String("exp_id", "test", "Experiment's ID, default value is test")
	target      = flag.String("target", "", "function's ip and port separated as host:port. There's no default value and should not start with http")
	nReqs       = flag.Int64("nreqs", 10, "number of requests, default 10000")
	lambda      = flag.Float64("lambda", 0.0, "Poisson's lambda value. Lambda 0 means sequential workload, default 0")
	resultsPath = flag.String("results_path", "", "absolute path for save results made. It has no default value")
)

func main() {
	flag.Parse()

	if err := checkWorkloadFlags(); err != nil {
		log.Fatalf("invalid flags: %v", err)
	}

	output := make([]string, *nReqs+1)
	output[0] = fmt.Sprintf("id,status,response_time,body,tsbefore,tsafter")

	fmt.Println("RUNNING WORKLOAD...")
	if *lambda == 0 {
		if err := sequentialWorkload(*target, *nReqs, output); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := poissonWorkload(*target, *nReqs, *lambda, output); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("SAVING RESULTS...")
	if err := createCsv(output, *resultsPath, *expId); err != nil {
		log.Fatal(err)
	}

}

func checkWorkloadFlags() error {
	if len(*expId) <= 0 {
		return fmt.Errorf("expID must exist. expID: %s", *expId)
	}
	if len(*target) <= 0 {
		return fmt.Errorf("target must exist. target: %s", *target)
	}
	if *nReqs <= 0 {
		return fmt.Errorf("nReqs must be bigger than zero. nReqs: %d", *nReqs)
	}
	if _, err := os.Stat(*resultsPath); os.IsNotExist(err) {
		return fmt.Errorf("resultsPath must exist. resultsPath: %s", *resultsPath)
	}

	return nil
}

func sequentialWorkload(target string, nReqs int64, output []string) error {
	for i := int64(1); i <= nReqs; i++ {
		status, responseTime, body, tsbefore, tsafter, err := sendReq(target)
		if err != nil {
			return err
		}

		body, err = getValueFromBodyMessage(body)
		if err != nil {
			return err
		}

		output[i] = fmt.Sprintf("%d,%d,%d,%s,%d,%d", i, status, responseTime, body, tsbefore, tsafter)
		if status != 200 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}

type poissonInterArrival struct {
	p *distuv.Poisson
}

func (pia *poissonInterArrival) next() float64 {
	return pia.p.Rand()
}

type InterArrival interface {
	next() float64
}

func NewPoissonInterArrival(lambda float64) InterArrival {
	return &poissonInterArrival{
		&distuv.Poisson{
			Lambda: lambda,
			Src:    rand.NewSource(uint64(time.Now().Nanosecond())),
		}}
}

func poissonWorkload(target string, nReqs int64, lambda float64, output []string) error {
	p := NewPoissonInterArrival(lambda)
	var wg sync.WaitGroup
	ch := make(chan string, nReqs+1)
	for i := int64(1); i <= nReqs; i++ {
		wg.Add(1)
		ia := p.next()
		time.Sleep(time.Duration(ia) * time.Millisecond)
		go func(id int64, wg *sync.WaitGroup, ch chan string) {
			defer wg.Done()

			status, responseTime, body, tsbefore, tsafter, err := sendReq(target)
			if err != nil {
				panic(err)
			}
			body, err = getValueFromBodyMessage(body)
			if err != nil {
				panic(err)
			}

			ch <- fmt.Sprintf("%d,%d,%d,%s,%d,%d", id, status, responseTime, body, tsbefore, tsafter)

			if status != 200 {
				time.Sleep(10 * time.Millisecond)
			}

		}(i, &wg, ch)
	}
	wg.Wait()
	chLen := len(ch)
	for i := 1; i <= chLen; i++ {
		output[i] = <-ch
	}
	close(ch)
	return nil
}

func getValueFromBodyMessage(body string) (string, error) {
	var bodyMapped map[string]interface{}
	err := json.Unmarshal([]byte(body), &bodyMapped)
	if err != nil {
		return "", fmt.Errorf("Couldn't execute unmarshal of body (%s), error (%v)", body, err.Error())
	}
	msg := bodyMapped["message"].(string)
	return msg, nil
}

func sendReq(target string) (int, int64, string, int64, int64, error) {
	before := time.Now()
	resp, err := http.Get(target)
	if err != nil {
		return 0, 0, "", 0, 0, err
	}
	defer resp.Body.Close()
	after := time.Now()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, "", 0, 0, err
	}
	status := resp.StatusCode
	body := string(bodyBytes)
	tsbefore := before.UnixNano()
	tsafter := after.UnixNano()
	responseTime := tsafter - tsbefore
	return status, responseTime, body, tsbefore, tsafter, nil
}

func createCsv(output []string, resultsPath, fileName string) error {
	file, err := os.OpenFile(resultsPath+fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	datawriter := bufio.NewWriter(file)
	for _, data := range output {
		_, _ = datawriter.WriteString(data + "\n")
	}
	datawriter.Flush()
	file.Close()
	return nil
}
