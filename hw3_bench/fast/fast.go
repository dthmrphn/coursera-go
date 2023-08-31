package fast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

//easyjson:json
type User struct {
	Browsers []string `json:"browsers,nocopy,omitempty"`
	Company  string   `json:"company,nocopy,omitempty"`
	Country  string   `json:"country,nocopy,omitempty"`
	Email    string   `json:"email,nocopy,omitempty"`
	Job      string   `json:"job,nocopy,omitempty"`
	Name     string   `json:"name,nocopy,omitempty"`
	Phone    string   `json:"phone,nocopy,omitempty"`
}

func EasyJson(w io.Writer, data []byte) {
	seen := map[string]bool{}
	user := User{}

	fmt.Fprintln(w, "found users:")

	for i, l := range bytes.Split(data, []byte("\n")) {
		err := user.UnmarshalJSON(l)
		if err != nil {
			panic(err)
		}

		android := false
		msie := false

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") {
				android = true
			} else if strings.Contains(browser, "MSIE") {
				msie = true
			} else {
				continue
			}

			seen[browser] = true
		}

		if !(android && msie) {
			continue
		}

		mail := strings.Replace(user.Email, "@", " [at] ", 1)
		fmt.Fprintf(w, "[%d] %s <%s>\n", i, user.Name, mail)
	}

	fmt.Fprintln(w, "\nTotal unique browsers", len(seen))
}

func Default(w io.Writer, data []byte) {
	seen := map[string]bool{}
	user := User{}

	fmt.Fprintln(w, "found users:")

	for i, l := range bytes.Split(data, []byte("\n")) {
		err := json.Unmarshal(l, &user)
		if err != nil {
			panic(err)
		}

		android := false
		msie := false

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") {
				android = true
			} else if strings.Contains(browser, "MSIE") {
				msie = true
			} else {
				continue
			}

			seen[browser] = true
		}

		if !(android && msie) {
			continue
		}

		mail := strings.Replace(user.Email, "@", " [at] ", 1)
		fmt.Fprintf(w, "[%d] %s <%s>\n", i, user.Name, mail)
	}

	fmt.Fprintln(w, "\nTotal unique browsers", len(seen))
}
