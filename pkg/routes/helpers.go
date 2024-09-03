package routes

import "net/url"

// AddQueryParam takes a URL, key, and value and returns the URL with the added query parameter.
func AddQueryParam(urlStr, key, value string) (string, error) {
	// Parse the URL to get a url.URL struct.
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Create a Values object from the parsed URL's query string.
	values := parsedURL.Query()

	// Add the key-value pair to the URL query parameters.
	values.Add(key, value)

	// Encode the query parameters and assign it back to the URL's RawQuery.
	parsedURL.RawQuery = values.Encode()

	// Return the updated URL as a string.
	return parsedURL.String(), nil
}
