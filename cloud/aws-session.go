package cloud

import (
	"bufio"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"os"
	"strings"
)

// AWSSession creates a new session struct to connect to AWS.
// Looks for credentials file in AWS_CREDENTIALS_FILE and ~/.aws/credentials.
func AWSSession() (*session.Session, error) {
	var sess *session.Session
	// Check if we were given the path to credentials file
	if credsPath := os.Getenv("AWS_CREDENTIALS_FILE"); credsPath != "" {
		// Create access key and secret access key vars
		accessKey := ""
		secretAccessKey := ""
		// Open the credentials file
		file, err := os.Open(credsPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		// Create a new buffered reader
		reader := bufio.NewReader(file)
		// Skip first line
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// Read second line
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// Strip whitespace
		line = strings.Replace(line, " ", "", -1)
		line = strings.Replace(line, "\n", "", -1)
		line = strings.Replace(line, "\r", "", -1)
		// Extract key
		if strings.Contains(line, "aws_access_key_id") {
			// Split string into KEY=VALUE
			ss := strings.Split(line, "=")
			// Set the access key to VALUE in KEY=VALUE
			accessKey = ss[1]
			// Make sure a plausibly valid key was extracted
			if len(accessKey) < 16 {
				return nil, errors.New("bad access key")
			}
		} else if strings.Contains(line, "aws_secret_access_key") {
			// Split string into KEY = VALUE
			ss := strings.Split(line, "=")
			// Set the secret access key to VALUE in KEY=VALUE
			secretAccessKey = ss[1]
			// Make sure a plausibly valid key was extracted
			if len(secretAccessKey) < 16 {
				return nil, errors.New("bad secret access key")
			}
		}
		// Read third line
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// Strip whitespace
		line = strings.Replace(line, " ", "", -1)
		line = strings.Replace(line, "\n", "", -1)
		line = strings.Replace(line, "\r", "", -1)
		// Extract key
		if strings.Contains(line, "aws_access_key_id") {
			// Split string into KEY=VALUE
			ss := strings.Split(line, "=")
			// Set the access key to VALUE in KEY=VALUE
			accessKey = ss[1]
			// Make sure a plausibly valid key was extracted
			if len(accessKey) < 16 {
				return nil, errors.New("bad access key")
			}
		} else if strings.Contains(line, "aws_secret_access_key") {
			// Split string into KEY = VALUE
			ss := strings.Split(line, "=")
			// Set the secret access key to VALUE in KEY=VALUE
			secretAccessKey = ss[1]
			// Make sure a plausibly valid key was extracted
			if len(secretAccessKey) < 16 {
				return nil, errors.New("bad secret access key")
			}
		}
		// Create new credentials
		creds := credentials.NewStaticCredentials(accessKey, secretAccessKey, "")
		sess, err = session.NewSession(&aws.Config{
			Region:      aws.String("us-west-1"),
			Credentials: creds,
		})
		if err != nil {
			return nil, err
		}
		return sess, nil
	}
	// Check if credentials file exists ~/.aws/credentials
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	credsFilePath := filepath.Join(usr.HomeDir, ".aws/credentials")
	if _, err := os.Stat(credsFilePath); os.IsNotExist(err) {
		return nil, errors.New("could not obtain AWS credenitals")
	}
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return sess, nil
}
