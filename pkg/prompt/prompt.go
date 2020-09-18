package prompt

import "github.com/AlecAivazis/survey/v2"

func Confirm(message string) bool {
	c := &survey.Confirm{
		Message: message,
	}

	var ans bool
	if err := survey.AskOne(c, &ans); err != nil {
		return false
	}

	return ans
}
