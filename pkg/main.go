package pkg

import (
	"fmt"
	"os"
)

func Execute(groupName, apiUsername, apiKey, apiEndpoint string) {

	discourse := DiscourseClient{
		Endpoint:    apiEndpoint,
		ApiUsername: apiUsername,
		ApiKey:      apiKey,
	}

	groupMembers, err := discourse.GetGroupMembers(groupName)

	if err != nil {
		fmt.Printf("Something went wrong while fetching members from the group %q\n", groupName)
		fmt.Println(err)
		os.Exit(1)
	}

	results := map[string]string{}

	for _, member := range groupMembers {
		email, err := discourse.GetUserEmail(member)

		if err != nil {
			fmt.Println(err)
			fmt.Printf("skipping user %q\n", member)
			continue
		}
		results[member] = email

	}

	output := ""
	fmt.Printf("%d email found for the %d members of the group %q\n", len(results), len(groupMembers), groupName)
	// Display username,email
	for id := range results {
		output = output + "\n" + fmt.Sprintf("%s,%s", id, results[id])
	}

	fmt.Println("====")
	fmt.Printf("Results: username,email\n%v\n", output)
	fmt.Println("====")

	// Display email
	for id := range results {
		output = output + "\n" + fmt.Sprintf("%s", results[id])
	}
	fmt.Println("====")
	fmt.Printf("Results: email\n%v\n", output)
	fmt.Println("====")
}
