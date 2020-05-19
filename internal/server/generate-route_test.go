package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"
)

// TestHandleGenerate_Basic tests the end product PDF of a generate request.
func TestHandleGenerate_Basic(t *testing.T) {
	err := os.Chdir("../../testing")
	if err != nil {
		t.Fatalf("error while moving into testing directory: %+v", err)
	}
	type test struct {
		Name string
		// Name of .tex file in the testing tex assets folder
		TexFile string
		// Name of .json file in the testing details assets folder
		DtlsFile string
		// List of resource file names in the testing resources assets folder
		Resources []string
		// Name of the .pdf file in the testing pdf assets folder to test final product against
		Expectation string
	}
	tt := []test{
		test{
			Name:        "first",
			TexFile:     "hello-world.tex",
			DtlsFile:    "hello-world_alice.json",
			Resources:   nil,
			Expectation: "hello-world_alice.pdf",
		},
	}
	for _, tc := range tt {
		// Each test case uses a new server
		s := Server{
			cmd:        "pdflatex",
			errLog:     log.New(log.Writer(), tc.Name+" Error: ", log.LstdFlags),
			infoLog:    log.New(ioutil.Discard, "", log.LstdFlags),
			tCacheSize: 1,
			rCacheSize: 1,
		}

		// Construct the request to the handler from the test case
		path := "./assets/templates/" + tc.TexFile
		tmplString, err := GetContentsBase64(path)
		if err != nil {
			t.Fatalf("error while opening template file: %+v", err)
		}
		var dtlsMap map[string]interface{}
		if tc.DtlsFile != "" {
			path = "./assets/details/" + tc.DtlsFile
			dtlsMap, err = GetContentsJSON(path)
			if err != nil {
				t.Fatalf("error while opening details file: %+v", err)
			}
		}
		resources := make(map[string]string)
		for _, rn := range tc.Resources {
			path = "./assets/resources/" + rn
			resource, err := GetContentsBase64(path)
			if err != nil {
				t.Fatalf("error while opening resource file: %+v", err)
			}
			resources[rn] = resource
		}
		testPayload, err := json.Marshal(struct {
			Template  string                 `json:"template"`
			Details   map[string]interface{} `json:"details"`
			Resources map[string]string      `json:"resources"`
		}{
			Template:  tmplString,
			Details:   dtlsMap,
			Resources: resources,
		})
		if err != nil {
			t.Fatalf("error while creating request payload: %+v", err)
		}
		req := httptest.NewRequest("GET", "/generate", bytes.NewBuffer(testPayload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Create the HTTP handler to be tested and save current working directory to move back into
		// after handler being tested is called; this is necessary since the handler changes the current working directory.
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("error while grabbing current directory: %+v", err)
		}
		hgFunc, err := s.handleGenerate()
		if err != nil {
			t.Fatalf("error while creating the function being tested: %+v", err)
		}
		hgFunc(rr, req)
		response := rr.Result()
		if response.StatusCode != 200 {
			t.Fatalf("Got non 200 status from result: %s", response.Status)
		}
		err = os.Chdir(wd)
		if err != nil {
			t.Fatalf("error while moving back into testing directory")
		}

		// Grab expected PDF to test against and compare it to the received PDF
		path = "./assets/PDFs/" + tc.Expectation
		expectedPDF, err := GetContentsBase64(path)
		if err != nil {
			t.Fatalf("error while reading expected PDF: %+v", err)
		}
		receivedPDF, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("error while reading received PDF: %+v", err)
		}
		response.Body.Close()
		receivedPDF64 := base64.StdEncoding.EncodeToString(receivedPDF)

		// Since PDFs seem to have some 'wiggle' to them, we have to make do with checking if our PDFs are 'close enough'
		// (We define 'close enough' as no more than 1% difference when comparing byte-to-byte)
		errorRate := DiffP(receivedPDF64, expectedPDF, t)
		if errorRate > 1.0 {
			t.Errorf("mismatch between received pdf and expected pdf exceeded 1%%: %f%%", errorRate)
		}
	}
}

func GetContentsBase64(path string) (string, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return "", err
	}
	fbytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	estring := base64.StdEncoding.EncodeToString(fbytes)
	return estring, nil
}

func GetContentsJSON(path string) (map[string]interface{}, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	err = json.NewDecoder(f).Decode(&data)
	return data, err
}

// DiffP tests the equality of the two strings and returns the percentage by which they differ.
func DiffP(received, expected string, t *testing.T) float32 {
	if len(received) != len(expected) {
		t.Fatalf("Received PDF differs from expected PDF: received length = %d \t expected length = %d",
			len(received), len(expected))
	}
	var mismatches int
	for i, c := range received {
		if byte(c) != byte(expected[i]) {
			mismatches++
		}
	}
	errorRate := float32(mismatches) / float32(len(expected))
	errorRate *= 100
	return errorRate
}
