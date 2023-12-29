package lasm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const (
	AUTH_URL                = "https://smartmeter.netz-noe.at/orchestration/Authentication/Login"
	BASIC_INFO_URL          = "https://smartmeter.netz-noe.at/orchestration/User/GetBasicInfo"
	ACCOUNT_INFO_URL        = "https://smartmeter.netz-noe.at/orchestration/User/GetAccountIdByBussinespartnerId?context=2"
	METERING_POINT_INFO_URL = "https://smartmeter.netz-noe.at/orchestration/User/GetMeteringPointByAccountId?accountId=%s&context=2"
	CONSUMPTION_DAY_URL     = "https://smartmeter.netz-noe.at/orchestration/ConsumptionRecord/Day?meterId=%s&day=%s"
	CONSUMPTION_MONTH_URL   = "https://smartmeter.netz-noe.at/orchestration/ConsumptionRecord/Month?meterId=%s&year=%d&month=%d"
	CONSUMPTION_YEAR_URL    = "https://smartmeter.netz-noe.at/orchestration/ConsumptionRecord/Year?meterId=%s&year=%d"
)

type LasmRegisterDate time.Time
type ValidFromDate time.Time
type MeterValueDate time.Time

type RelationType string

const (
	FROM_GRID RelationType = "Bezug"
	TO_GRID   RelationType = "Einspeisung"
)

func (e *LasmRegisterDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02T15:04:05.99", s)
	if err != nil {
		return err
	}
	*e = LasmRegisterDate(t)
	return nil
}

func (v *ValidFromDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	*v = ValidFromDate(t)
	return nil
}

func (v *MeterValueDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	*v = MeterValueDate(t)
	return nil
}

// BasicInfoResponse encapsulates basic account information.
type BasicInfoResponse struct {
	GpNumber      string           `json:"gpNummer"`
	TitlePre      string           `json:"titelVorgestellt"`
	TitlePost     string           `json:"titelNachgestellt"`
	FormOfAddress string           `json:"anrede"`
	Name          string           `json:"vorname"`
	Surname       string           `json:"nachname"`
	RegisterDate  LasmRegisterDate `json:"registerDate"`
	From          string           `json:"von"`
}

// AccountInfoResponse encapsulates detailed info about a specific endpoint.
type AccountInfoResponse struct {
	GpNumber            string `json:"gpNumber"`
	AccountID           string `json:"accountId"`
	ExternalPowerSupply bool   `json:"externalPowerSupply"`
	HasSmartMeter       bool   `json:"hasSmartMeter"`
	HasElectricity      bool   `json:"hasElectricity"`
	HasGas              bool   `json:"hasGas"`
	IsCommunicative     bool   `json:"hasCommunicative"`
	HasOptIn            bool   `json:"hasOptIn"`
	IsActive            bool   `json:"hasActive"`
}

// MeterInfoResponse encapsulates information about a specific metering point.
type MeterInfoResponse struct {
	Id                          string        `json:"meteringPointId"`
	TypeOfRelation              RelationType  `json:"typeOfRelation"`
	FtmReadOut                  bool          `json:"ftmReadOut"`
	FtmReadOutProvider          bool          `json:"ftmReadOutProvider"`
	CommunityProductionFacility bool          `json:"communityProductionFacility"`
	HasFtmMeterData             bool          `json:"hasFtmMeterData"`
	ValidFrom                   ValidFromDate `json:"validFrom"`
	SmartMeterType              string        `json:"smartMeterType"`
	Locked                      bool          `json:"locked"`
	PointOfConsumption          string        `json:"pointOfConsumption"`
	Category                    string        `json:"category"`
}

// MeterValuesResponseDayMonth is used in daily and monthly queries.
type MeterValuesResponseDayMonth struct {
	MeteredValues      []float64        `json:"meteredValues"`
	MeteredPeakDemands []float64        `json:"meteredPeakDemands"`
	PeakDemandTimes    []MeterValueDate `json:"peakDemandTimes"`
}

// MeterValuesResonseYear is used when querying data for a whole year.
type MeterValuesResponseYear struct {
	Values          []float64        `json:"values"`
	PeakDemands     []float64        `json:"peakDemands"`
	PeakDemandTimes []MeterValueDate `json:"peakDemandTimes"`
}

type MeterValue struct {
	Timestamp    time.Time
	MeteredValue float64
}

// LasmClient represents an Lasm endpoint
type LasmClient struct {
	client http.Client
}

// NewLasmClient creates a new client, which is used for querying account and data endpoints
func NewLasmClient() *LasmClient {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	return &LasmClient{
		client: *client,
	}
}

// Login is used to initiate a session with the smartmeter portal API.
func (c *LasmClient) Login(username, password string) error {
	postBody, _ := json.Marshal(map[string]string{
		"user": username,
		"pwd":  password,
	})
	requestBody := bytes.NewBuffer(postBody)

	resp, err := c.client.Post(AUTH_URL, "application/json", requestBody)
	if err != nil {
		return fmt.Errorf("could not login: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed, status code: %d", resp.StatusCode)
	}

	return nil
}

// GetBasicInfo returns basic information about the account.
func (c *LasmClient) GetBasicInfo() (BasicInfoResponse, error) {
	var nilValue BasicInfoResponse

	basicInfoResponse, err := c.client.Get(BASIC_INFO_URL)
	if err != nil {
		return nilValue, fmt.Errorf("could not retrieve basic info: %s", err)
	}
	defer basicInfoResponse.Body.Close()

	if basicInfoResponse.StatusCode != http.StatusOK {
		return nilValue, fmt.Errorf("could not retrieve basic info, status code: %d", basicInfoResponse.StatusCode)
	}

	basicInfoPayload, err := io.ReadAll(basicInfoResponse.Body)
	if err != nil {
		return nilValue, fmt.Errorf("could not read basic info payload: %s", err)
	}

	var basicInfo BasicInfoResponse
	err = json.Unmarshal(basicInfoPayload, &basicInfo)
	if err != nil {
		return nilValue, fmt.Errorf("could not unmarshal basic info payload: %s", err)
	}

	return basicInfo, nil
}

// GetAccountInfos returns a list of AccountInfo information, which encapsulates different metering points,
// e.g. for energy consumption and delivery,
func (c *LasmClient) GetAccountInfos() ([]AccountInfoResponse, error) {
	var nilValue []AccountInfoResponse

	accountInfoResponse, err := c.client.Get(ACCOUNT_INFO_URL)
	if err != nil {
		return nilValue, fmt.Errorf("could not retrieve account info: %s", err)
	}
	defer accountInfoResponse.Body.Close()

	if accountInfoResponse.StatusCode != http.StatusOK {
		return nilValue, fmt.Errorf("could not retrieve account info, status code: %d", accountInfoResponse.StatusCode)
	}

	accountInfoPayload, err := io.ReadAll(accountInfoResponse.Body)
	if err != nil {
		return nilValue, fmt.Errorf("could not read account info payload: %s", err)
	}

	var accountInfos []AccountInfoResponse
	err = json.Unmarshal(accountInfoPayload, &accountInfos)
	if err != nil {
		return nilValue, fmt.Errorf("could not unmarshal account info payload: %s", err)
	}

	return accountInfos, nil
}

// GetMeterInfos returns the list of metering points to a specific account.
func (c *LasmClient) GetMeterInfos(accountId string) ([]MeterInfoResponse, error) {
	var nilValue []MeterInfoResponse

	meterRequestUrl := fmt.Sprintf(METERING_POINT_INFO_URL, accountId)

	meterInfoResponse, err := c.client.Get(meterRequestUrl)
	if err != nil {
		return nilValue, fmt.Errorf("could not retrieve meter info: %s", err)
	}
	defer meterInfoResponse.Body.Close()

	if meterInfoResponse.StatusCode != http.StatusOK {
		return nilValue, fmt.Errorf("could not retrieve meter info, status code: %d", meterInfoResponse.StatusCode)
	}

	meterInfoPayload, err := io.ReadAll(meterInfoResponse.Body)
	if err != nil {
		return nilValue, fmt.Errorf("could not read meter info payload: %s", err)
	}

	var meterInfos []MeterInfoResponse
	err = json.Unmarshal(meterInfoPayload, &meterInfos)
	if err != nil {
		return nilValue, fmt.Errorf("could not unmarshal meter info payload: %s", err)
	}

	return meterInfos, nil
}

// GetConsumptionByMeterAndDate returns a list of values for a specific day, typically in a 15 minutes interval.
func (c *LasmClient) GetConsumptionByMeterAndDate(meterId string, date time.Time) (MeterValuesResponseDayMonth, []MeterValue, error) {
	var nilValue MeterValuesResponseDayMonth

	dateFormatted := date.Format("2006-01-02")

	dataRequestUrl := fmt.Sprintf(CONSUMPTION_DAY_URL, meterId, dateFormatted)

	// data will be returned as multiple arrays of length 96 (four 15 minutes intervalls per hour)
	meterValuesResponse, err := c.client.Get(dataRequestUrl)
	if err != nil {
		return nilValue, nil, fmt.Errorf("could not retrieve meter info: %s", err)
	}
	defer meterValuesResponse.Body.Close()

	if meterValuesResponse.StatusCode != http.StatusOK {
		return nilValue, nil, fmt.Errorf("could not retrieve meter info, status code: %d", meterValuesResponse.StatusCode)
	}

	meterValuesPayload, err := io.ReadAll(meterValuesResponse.Body)
	if err != nil {
		return nilValue, nil, fmt.Errorf("could not read meter info payload: %s", err)
	}

	var meterValues MeterValuesResponseDayMonth
	err = json.Unmarshal(meterValuesPayload, &meterValues)
	if err != nil {
		return nilValue, nil, fmt.Errorf("could not unmarshal meter info payload: %s", err)
	}

	values := make([]MeterValue, len(meterValues.MeteredValues))
	interval := time.Minute * 15
	ts := time.Date(date.Year(), date.Month(), date.Day(), 0, 15, 0, 0, time.UTC)

	for i, value := range meterValues.MeteredValues {
		newValue := MeterValue{
			Timestamp:    ts,
			MeteredValue: value,
		}
		values[i] = newValue
		ts = ts.Add(interval)
	}

	return meterValues, values, nil
}

// GetConsumptionByMeterAndYearAndMonth returns a list of daily values for a specific month of a year
func (c *LasmClient) GetConsumptionByMeterAndYearAndMonth(meterId string, date time.Time) (MeterValuesResponseDayMonth, error) {
	var nilValue MeterValuesResponseDayMonth

	year := date.Year()
	month := date.Month()

	dataRequestUrl := fmt.Sprintf(CONSUMPTION_MONTH_URL, meterId, year, month)

	// data will be returned as multiple arrays of length 30/31 (tuple for each day)
	meterValuesResponse, err := c.client.Get(dataRequestUrl)
	if err != nil {
		return nilValue, fmt.Errorf("could not retrieve meter info: %s", err)
	}
	defer meterValuesResponse.Body.Close()

	if meterValuesResponse.StatusCode != http.StatusOK {
		return nilValue, fmt.Errorf("could not retrieve meter info, status code: %d", meterValuesResponse.StatusCode)
	}

	meterValuesPayload, err := io.ReadAll(meterValuesResponse.Body)
	if err != nil {
		return nilValue, fmt.Errorf("could not read meter info payload: %s", err)
	}

	var meterValues MeterValuesResponseDayMonth
	err = json.Unmarshal(meterValuesPayload, &meterValues)
	if err != nil {
		return nilValue, fmt.Errorf("could not unmarshal meter info payload: %s", err)
	}

	return meterValues, nil
}

// GetConsumptionByMeterAndDate returns a list of monthly values for a specific year
func (c *LasmClient) GetConsumptionByMeterAndYear(meterId string, date time.Time) (MeterValuesResponseYear, error) {
	var nilValue MeterValuesResponseYear

	year := date.Year()

	dataRequestUrl := fmt.Sprintf(CONSUMPTION_YEAR_URL, meterId, year)

	// data will be returned as multiple arrays of length 12 (tuple for each month)
	meterValuesResponse, err := c.client.Get(dataRequestUrl)
	if err != nil {
		return nilValue, fmt.Errorf("could not retrieve meter info: %s", err)
	}
	defer meterValuesResponse.Body.Close()

	if meterValuesResponse.StatusCode != http.StatusOK {
		return nilValue, fmt.Errorf("could not retrieve meter info, status code: %d", meterValuesResponse.StatusCode)
	}

	meterValuesPayload, err := io.ReadAll(meterValuesResponse.Body)
	if err != nil {
		return nilValue, fmt.Errorf("could not read meter info payload: %s", err)
	}

	var meterValues MeterValuesResponseYear
	err = json.Unmarshal(meterValuesPayload, &meterValues)
	if err != nil {
		return nilValue, fmt.Errorf("could not unmarshal meter info payload: %s", err)
	}

	return meterValues, nil
}
