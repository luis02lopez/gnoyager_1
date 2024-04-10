package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQLResponse struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

var transactionsCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "gno_transactions_total",
		Help: "Total number of transactions.",
	},
)

var messageTypeCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "gno_message_types_total",
		Help: "Total number of each message type.",
	},
	[]string{"typeUrl"},
)

var senderCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "gno_senders_total",
		Help: "Total number of transactions by each sender.",
	},
	[]string{"sender"},
)

var packagesDeployedCounter = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "gno_packages_deployed_total",
        Help: "Total number of packages deployed.",
    },
)

var activePackagesCounter = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "gno_active_packages_total",
        Help: "Total number of calls to each package.",
    },
    []string{"pkg_path"},
)

var latestBlockHeightGauge = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Name: "gno_latest_block_height",
        Help: "The latest block height.",
    },
)

func init() {
	prometheus.MustRegister(transactionsCounter)
	prometheus.MustRegister(messageTypeCounter)
	prometheus.MustRegister(senderCounter)
	prometheus.MustRegister(packagesDeployedCounter)
	prometheus.MustRegister(activePackagesCounter)
	prometheus.MustRegister(latestBlockHeightGauge)
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		fmt.Println("Application is running and listening on http://localhost:2112/metrics")
		http.ListenAndServe(":2112", nil)
	}()

	go func() {
		for {
			pollGraphQL()
			// time.Sleep(1 * time.Minute)
			time.Sleep(5 * time.Second)
			// senderCounter.Reset()
		}
	}()

	select {} // block forever
}

var processedTransactions = make(map[string]bool)

func pollGraphQL() {

	url := "http://gno_indexer:8546/graphql/query"
	data := GraphQLRequest{
		Query: `{ transactions(filter: {}) { index hash block_height gas_used memo messages { typeUrl route value { __typename ... on BankMsgSend { from_address amount } ... on MsgCall { caller send pkg_path func } ... on MsgAddPackage { creator package { name path } } ... on MsgRun { caller package { name path } } } } } }`,
	}

	pollLatestBlockHeight()

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error on encoding data to JSON.\n[ERROR] -", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error on response.\n[ERROR] -", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error while reading the response bytes:", err)
		return
	}

	var response GraphQLResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error while parsing the JSON response:", err)
		return
	}

	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error while converting data to map[string]interface{}")
		return
	}

	transactions, ok := dataMap["transactions"].([]interface{})
	if !ok {
		fmt.Println("Waiting on supernova to run transactions...")
		// fmt.Println("Error while converting transactions to []interface{}")
		return
	}
	fmt.Println("Collecting metrics...")

	for _, transaction := range transactions {
		transactionMap, ok := transaction.(map[string]interface{})
		if !ok {
			fmt.Println("Error while converting transaction to map[string]interface{}")
			continue
		}

        hash, ok := transactionMap["hash"].(string)
        if !ok {
            fmt.Println("Error while converting hash to string")
            continue
        }

        if processedTransactions[hash] {
            // Skip this transaction because it has already been processed
            continue
        }

        // Mark this transaction as processed
        processedTransactions[hash] = true

		transactionsCounter.Inc()

		messages, ok := transactionMap["messages"].([]interface{})
		if !ok {
			fmt.Println("Error while converting messages to []interface{}")
			continue
		}

		for _, message := range messages {
			messageMap, ok := message.(map[string]interface{})
			if !ok {
				fmt.Println("Error while converting message to map[string]interface{}")
				continue
			}

			typeUrl, ok := messageMap["typeUrl"].(string)
			if !ok {
				fmt.Println("Error while converting typeUrl to string")
				continue
			}

			messageTypeCounter.With(prometheus.Labels{"typeUrl": typeUrl}).Inc()

			value, ok := messageMap["value"].(map[string]interface{})
			if !ok {
				fmt.Println("Error while converting value to map[string]interface{}")
				continue
			}

			typename, ok := value["__typename"].(string)
			if !ok {
				fmt.Println("Error while converting __typename to string")
				continue
			}

			var sender string
			switch typename {
			case "BankMsgSend":
				sender, ok = value["from_address"].(string)
			case "MsgCall", "MsgRun":
				sender, ok = value["caller"].(string)
			case "MsgAddPackage":
				sender, ok = value["creator"].(string)
			}

			if !ok {
				fmt.Println("Error while converting sender to string")
				continue
			}

			senderCounter.With(prometheus.Labels{"sender": sender}).Inc()
			
			if typename == "MsgAddPackage" {
                packagesDeployedCounter.Inc()
            }

            if typename == "MsgCall" {
                pkgPath, ok := value["pkg_path"].(string)
                if !ok {
                    fmt.Println("Error while converting pkg_path to string")
                    continue
                }

                activePackagesCounter.With(prometheus.Labels{"pkg_path": pkgPath}).Inc()
            }
		}
	}

	// fmt.Println("Response data:", response.Data)
	if len(response.Errors) > 0 {
		fmt.Println("Response errors:", response.Errors)
	}
}

func pollLatestBlockHeight() {
    url := "http://gno_indexer:8546/graphql/query"
    data := GraphQLRequest{
        Query: `{ latestBlockHeight }`,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        fmt.Println("Error on encoding data to JSON.\n[ERROR] -", err)
        return
    }

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Println("Error on response.\n[ERROR] -", err)
        return
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Error while reading the response bytes:", err)
        return
    }

    var response GraphQLResponse
    err = json.Unmarshal(body, &response)
    if err != nil {
        fmt.Println("Error while parsing the JSON response:", err)
        return
    }

    dataMap, ok := response.Data.(map[string]interface{})
    if !ok {
        fmt.Println("Error while converting data to map[string]interface{}")
        return
    }

    latestBlockHeight, ok := dataMap["latestBlockHeight"].(float64)
    if !ok {
        fmt.Println("Error while converting latestBlockHeight to float64")
        return
    }

    latestBlockHeightGauge.Set(latestBlockHeight)
}
