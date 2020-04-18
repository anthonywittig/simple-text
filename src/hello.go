package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

	message, err := getMessage()
	if err != nil {
		panic(fmt.Errorf("fatal error getting message: %s", err))
	}

	fmt.Printf("You are about to send the following message:\n\n%s\n\nAre you sure you want to (type YES if so).\n", message)
	shouldWeContinue, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(fmt.Errorf("fatal error user input: %s", err))
	}

	if shouldWeContinue != "YES\n" {
		panic("Sounds like we're not ready, exiting!")
	}

	attemptedNumbers := map[string]struct{}{}
	for _, c := range contacts {
		if _, ok := attemptedNumbers[c.PhoneNumber]; ok {
			continue
		}
		attemptedNumbers[c.PhoneNumber] = struct{}{}

		if err := sendMessage(message, c, config.Twilio); err != nil {
			errMsg := fmt.Sprintf("error sending message for %s (%s)", c.Name, c.PhoneNumber)
			fmt.Println(errors.Wrap(err, errMsg))
		} else {
			fmt.Printf("Sent message to %s (%s)\n", c.Name, c.PhoneNumber)
		}
	}
}

func sendMessage(message string, contact Contact, twilio Twilio) error {
	msgData := url.Values{}
	msgData.Set("To", contact.PhoneNumber)
	msgData.Set("From", twilio.PhoneNumber)
	msgData.Set("Body", message)
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

func getMessage() (string, error) {
	file, err := os.Open("message.txt")
	if err != nil {
		return "", err
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", b), nil
}
