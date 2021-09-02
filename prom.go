package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
)

// MetricResponse is root of JSON type
type MetricResponse struct {
	Data Data `json:"data,omitempty"`
}

// Data is 1st branch of JSON type
type Data struct {
	Results []Result `json:"result,omitempty"`
}

// Result is 2nd branch of JSON type
type Result struct {
	MetricInfo  map[string]string `json:"metric,omitempty"`
	MetricValue []interface{}     `json:"value,omitempty"` //Index 0 is unix_time, index 1 is sample_value (metric value)
}

func DecodeJsonDataToStruct(metrics *MetricResponse, resp *http.Response) {
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(metrics)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ConvertStringToFloat(metric MetricResponse) float64 {
	for _, result := range metric.Data.Results {
		rawData := fmt.Sprintf("%v", result.MetricValue[1])
		convertedData, err := strconv.ParseFloat(rawData, 64)
		if err != nil {
			fmt.Println(err)
		}
		return convertedData
	}
	return 0.0
}

func GetResouceInstance(instance string, resource string, result []string) ([]string, error) {
	var query string
	var domain = "192.168.191.252:9090/"
	var metrics MetricResponse
	switch resource {
	case "MEM":
		query = "((node_memory_MemTotal_bytes{job=\"node_exporter\"}-node_memory_MemAvailable_bytes{job=\"node_exporter\"}))/(1024*1024)"
	case "CPU":
		query = "100-(irate(node_cpu_seconds_total{mode=\"idle\",job=\"node_exporter\"}[10m])*100)"
		// query = "100-(avg by (instance, job) (irate(node_cpu_seconds_total{mode=\"idle\",job=\"node_exporter\"}[10m])*100))"
	default:
		fmt.Errorf("resource is not understandable")
	}
	resp, err := http.Get("http://" + domain + "api/v1/query?query=" + query + "")
	if err != nil {
		fmt.Println(err)
	}
	DecodeJsonDataToStruct(&metrics, resp)
	result = append(result, fmt.Sprintf("%v", metrics.Data.Results[0].MetricValue[0]))
	for _, v := range metrics.Data.Results {
		info := v.MetricInfo
		if info["instance"] == instance {
			result = append(result, fmt.Sprintf("%v", v.MetricValue[1]))
		}
	}
	fmt.Errorf("instance IP not found")
	return result, err
}

func WriteExcel(file *excelize.File, sheet string, row string, v interface{}) {
	var i = 0
	for true {
		i++
		index := strconv.Itoa(i)
		cvalue := file.GetCellValue(sheet, row+index)
		if cvalue != "" {
			continue
		} else {
			file.SetCellValue(sheet, row+index, v)
			if err := file.SaveAs("record.xlsx"); err != nil {
				fmt.Println(err)
			}
			break
		}
	}
}

func main() {
	const SERVER string = "192.168.101.34:9100"
	// open EXCEL file and categories
	// file, err := excelize.OpenFile("record.xlsx")
	file := excelize.NewFile()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	categories := map[string]string{"A1": "TimeMEM", "B1": "MEM", "C1": "TimeCPU", "D1": "CPU"}
	for k, v := range categories {
		file.SetCellValue("Sheet1", k, v)
	}

	for true { // keep recording resource until break command detected
		var resultMem []string
		var resultCPU []string

		// query RAM
		resultMem, err := GetResouceInstance(SERVER, "MEM", resultMem)
		if err != nil {
			fmt.Println(err)
		}
		timeMem, err1 := strconv.ParseFloat(resultMem[0], 64)
		if err1 != nil {
			fmt.Println(err1)
		}
		t := time.Unix(int64(timeMem), 0)
		fmt.Println("Time MEM: ", t.Format("2006-01-02 15:04:05"))
		mem, err2 := strconv.ParseFloat(resultMem[1], 64)
		if err2 != nil {
			fmt.Println(err2)
		}
		// query CPU
		resultCPU, err3 := GetResouceInstance(SERVER, "CPU", resultCPU)
		if err3 != nil {
			fmt.Println(err3)
		}
		timeCPU, err4 := strconv.ParseFloat(resultCPU[0], 64)
		if err4 != nil {
			fmt.Println(err4)
		}
		t1 := time.Unix(int64(timeCPU), 0)
		fmt.Println("Time CPU: ", t1.Format("2006-01-02 15:04:05"))
		sumCPU := 0.0
		for i := range resultCPU {
			if i == 0 {
				continue
			}
			cpu, err5 := strconv.ParseFloat(resultCPU[i], 64)
			if err5 != nil {
				fmt.Println(err5)
			}
			sumCPU += cpu
		}
		memRound := math.Round(mem*1000) / 1000
		cpuRound := math.Round(sumCPU*1000) / (1000 * float64(len(resultCPU)))
		// fmt.Println(reflect.TypeOf(mem))
		WriteExcel(file, "Sheet1", "A", t)
		WriteExcel(file, "Sheet1", "B", memRound)
		WriteExcel(file, "Sheet1", "C", t1)
		WriteExcel(file, "Sheet1", "D", cpuRound)
		fmt.Println("finish recording")
		// time.Sleep(500 * time.Millisecond) // turn on if program queries too fast
	}

}
