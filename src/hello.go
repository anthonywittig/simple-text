package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Contact struct {
	Name        string
	PhoneNumber string
}

type Config struct {
	Twilio Twilio
}

type Twilio struct {
	Account     string
	SID         string
	Secret      string
	PhoneNumber string
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("fatal error parsing config: %s", err))
	}

	contacts, err := getContacts()
	if err != nil {
		panic(fmt.Errorf("fatal error getting contacts: %s", err))
	}

	for _, c := range contacts {
		//fmt.Printf("%+v\n", c)
		if err := sendMessage(c, config.Twilio); err != nil {
			panic(fmt.Errorf("fatal error sending message: %s", err))
		}
	}
}

func sendMessage(contact Contact, twilio Twilio) error {
	msgData := url.Values{}
	msgData.Set("To", contact.PhoneNumber)
	msgData.Set("From", twilio.PhoneNumber)
	msgData.Set("Body", "test text - hi!")
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", twilio.Account)
	req, err := http.NewRequest("POST", apiURL, &msgDataReader)
	if err != nil {
		return err
	}

	req.SetBasicAuth(twilio.SID, twilio.Secret)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&data)
		if err != nil {
			return err
		}
		fmt.Printf("Send message to %s (%s)\n", contact.Name, contact.PhoneNumber)
	} else {
		return errors.New(fmt.Sprintf("bad status code: %s", resp.Status))
	}
	return nil
}

func getContacts() ([]Contact, error) {
	csvFile, err := os.Open("contacts.csv")
	if err != nil {
		return nil, err
	}

	contacts := []Contact{}

	r := csv.NewReader(csvFile)
	for {
		record, err := r.Read()
		if err == io.EOF {
			return contacts, nil
		} else if err != nil {
			return nil, err
		}

		phone, err := cleanPhone(record[1])
		if err != nil {
			log.Print(errors.Wrap(err, fmt.Sprintf("skipping record: %+v", record)))
			continue
		}

		contacts = append(
			contacts,
			Contact{
				Name:        record[0],
				PhoneNumber: phone,
			},
		)
	}
}

func cleanPhone(dirty string) (string, error) {
	original := dirty
	for _, c := range []string{" ", "(", ")", "-"} {
		dirty = strings.ReplaceAll(dirty, c, "")
	}
	if len(dirty) != 10 {
		return "", errors.New(fmt.Sprintf("phone number sucks: '%s' ('%s')", original, dirty))
	}

	clean, err := regexp.MatchString(`\d{10}`, dirty)
	if clean && err != nil {
		return "", errors.New(fmt.Sprintf("phone number sucks: '%s' ('%s')", original, dirty))
	}

	return fmt.Sprintf("+1%s", dirty), nil
}
