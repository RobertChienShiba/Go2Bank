package mail

import (
	"testing"

	"github.com/RobertChienShiba/simplebank/util"
	"github.com/stretchr/testify/require"
)

func TestSendEmailWithGmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	config, err := util.LoadConfig("..")
	require.NoError(t, err)

	sender := NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)

	subject := "A test email"
	content := `
	<h1>Hello world</h1>
	<p>This is a test message from <a href="https://www.google.com.tw/">Go2Bank</a></p>
	`
	to := []string{"bookly162@gmail.com"}
	attachFiles := []string{"../README.md"}

	err = sender.SendEmail(subject, content, to, attachFiles)
	require.NoError(t, err)
}
