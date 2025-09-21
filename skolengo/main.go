package main

import (
	"fmt"
	"io"
	"os"
	"time"

	skolengo "github.com/espcaa/skolen-go"
)

func main() {
	// Load tokens.json file
	file, err := os.Open("tokens.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	// Create the client
	client, err := skolengo.NewClientFromJSON(data)
	if err != nil {
		panic(err)
	}

	periodStart := time.Now()
	periodEnd := time.Now().AddDate(0, 0, 2)

	// Get the timetable
	timetable, err := client.GetTimetable(client.UserInfo.UserID, client.UserInfo.SchoolID, client.UserInfo.EMSCode, periodStart, periodEnd, 0)
	if err != nil {
		panic(err)
	}

	for _, day := range timetable {
		fmt.Println("Date:", day.Date)

		// Loop over lessons in that day
		for _, lesson := range day.Lessons {
			fmt.Println(" Lesson:", lesson.Subject.Label, "at", lesson.Location)
		}

		// Loop over assignments in that day
		for _, assign := range day.Assignments {
			fmt.Println(" Assignment:", assign.Title, "due", assign.DueDateTime)
		}
	}
}
