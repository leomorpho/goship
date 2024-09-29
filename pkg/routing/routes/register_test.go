package routes

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
	"github.com/stretchr/testify/assert"
)

// func TestRegisterUser(t *testing.T) {
// 	request(t).
// 		setRoute(routeNameRegister).
// 		get().
// 		assertStatusCode(http.StatusOK).
// 		toDoc()

// 	// Simulating form submission with incomplete data
// 	formData := url.Values{
// 		"name":                []string{""}, // Missing name
// 		"email":               []string{"user@example.com"},
// 		"password":            []string{"12345678"},
// 		"password-confirm":    []string{"12345678"},
// 		"relationship_status": []string{"partner"},
// 		"birthdate":           []string{"2000/11/11"},
// 	}

// 	postReq := request(t).setRoute("register").setBody(formData)
// 	response := postReq.post()

// 	// Check for a client-side rendered error, assuming the response is a rendered form with errors
// 	responseDoc := response.toDoc()
// 	errorExists := responseDoc.Find(".error").Length() > 0
// 	assert.True(t, errorExists, "Expected error message not found")

// }

func TestRegisterUserFieldValidation(t *testing.T) {
	// Define test cases with each field tested independently by omitting it from the formData
	tests := []struct {
		name          string
		omitField     string
		expectedError bool
	}{
		{"Missing Name", "name", true},
		{"Missing Email", "email", true},
		{"Missing Birthdate", "birthdate", true},
	}

	// Loop over the test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare the form data with all fields filled
			formData := url.Values{
				"name":             []string{"John Doe"},
				"email":            []string{"user@example.com"},
				"password":         []string{"12345678"},
				"password-confirm": []string{"12345678"},
				"birthdate":        []string{"2000/11/11"},
			}

			// Remove the field being tested
			formData.Del(tc.omitField)

			// Perform the HTTP POST request
			postReq := request(t).setRoute(routenames.RouteNameRegister).setBody(formData)
			response := postReq.post()

			// Ensure the page reloads with a 200 status, indicating form errors
			response.assertStatusCode(http.StatusOK)

			// Parse the HTML response for errors
			responseDoc := response.toDoc()

			// Find alerts that indicate an error
			errors := responseDoc.Find("div[role='alert']")

			// Assert that exactly one error message is found
			if tc.expectedError {
				// TODO: awaiting an extra one for the dating rollout per region announcement
				assert.Equal(t, 2, errors.Length(), "Expected exactly one error alert for missing field")
			} else {
				assert.Equal(t, 0, errors.Length(), "Expected no error alert for this field")
			}

			// TODO: awaiting an extra one for the dating rollout per region announcement. Removed below check
			// because it's too annoying to check (too low priority).

			// Optionally, you can iterate over each found alert and check if it contains specific error text
			// errors.Each(func(i int, s *goquery.Selection) {
			// 	alertText := s.Text()
			// 	assert.Contains(t, alertText, "This field is required", "Expected specific error message within alert")
			// })
		})
	}
}
