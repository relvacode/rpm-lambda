package setup

import "github.com/aws/aws-sdk-go/aws/session"

func NewSession() (*session.Session, error) {
	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	s.Config.Logger = NewLog("aws:session")
	return s, nil
}
