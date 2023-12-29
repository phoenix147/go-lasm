package main

import (
	"fmt"
	"lasm"
	"log"
	"os"
	"time"
)

func main() {

	username := os.Getenv("EVN_USR")
	password := os.Getenv("EVN_PWD")

	if username == "" || password == "" {
		log.Fatal("username and password must be supplied")
	}

	e := lasm.NewLasmClient()

	err := e.Login(username, password)
	if err != nil {
		log.Fatal(err)
	}

	basicInfo, err := e.GetBasicInfo()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", basicInfo)

	accountInfos, err := e.GetAccountInfos()
	if err != nil {
		log.Fatal(err)
	}

	for _, accountInfo := range accountInfos {
		meterInfos, err := e.GetMeterInfos(accountInfo.AccountID)
		if err != nil {
			log.Fatal(err)
		}

		for _, meterInfo := range meterInfos {
			_, values, err := e.GetConsumptionByMeterAndDate(meterInfo.Id, time.Now().Add(time.Hour*-24))
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Meter ID: %s, Relation: %s\n", meterInfo.Id, meterInfo.TypeOfRelation)

			for i, value := range values {
				fmt.Printf("%02d - %s: %f\n", i, value.Timestamp.String(), value.MeteredValue)
			}
		}
	}
}
