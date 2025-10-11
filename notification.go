package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	model "github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func send(webhookURL string, payload model.CommandResponse) error {
	marshalContent, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal payload")
	}
	var jsonStr = []byte(marshalContent)
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header.Set("X-Custom-Header", "aws-sns")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed tο send HTTP request")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("Failed to close response body")
		}
	}()
	return nil
}

func sendWhoIsOnCallNotification(primary, secondary string) error {
	fields := []*model.SlackAttachmentField{
		{Title: "Who is onCall?", Short: false},
		{Title: "Primary", Value: fmt.Sprintf("@%s", primary), Short: true},
	}

	// Only add secondary field if secondary is not empty
	if secondary != "" {
		fields = append(fields, &model.SlackAttachmentField{Title: "Secondary", Value: fmt.Sprintf("@%s", secondary), Short: true})
	}

	fields = append(fields, &model.SlackAttachmentField{Value: "_Who you gonna call?_ Use: @sreoncall", Short: false})

	attachment := &model.SlackAttachment{
		Title:     "SRE Oncall",
		TitleLink: "https://mattermost.app.opsgenie.com/alert",
		Color:     "#0000ff",
		Fields:    fields,
	}

	payload := model.CommandResponse{
		Username:    "OnCall Notifier",
		IconURL:     "https://vignette.wikia.nocookie.net/ghostbusters/images/a/a7/NoGhostSign.jpg/revision/latest/scale-to-width-down/340?cb=20090213041921",
		Attachments: []*model.SlackAttachment{attachment},
	}
	err := send(os.Getenv("MATTERMOST_SREONCALL_NOTIFICATION_HOOK"), payload)
	if err != nil {
		return errors.Wrap(err, "failed tο send Mattermost request payload")
	}
	return nil
}

func sendWhoIsSRESupportNotification(users []string) error {
	fields := []*model.SlackAttachmentField{
		{Title: "Who is on SRE Support?", Short: false},
		{Value: "_Who you gonna ask for help?_ Use: @sresupport", Short: false},
	}

	for _, user := range users {
		fields = append(fields, &model.SlackAttachmentField{Title: "Support A", Value: fmt.Sprintf("@%s", user), Short: true})
	}

	attachment := &model.SlackAttachment{
		Title:     "SRE Support",
		TitleLink: "https://mattermost.app.opsgenie.com/alert",
		Color:     "#0000ff",
		Fields:    fields,
	}

	payload := model.CommandResponse{
		Username:    "SRE Support Notifier",
		IconURL:     "https://mystickermania.com/cdn/stickers/memes/sticker_2094-512x512.png",
		Attachments: []*model.SlackAttachment{attachment},
	}
	err := send(os.Getenv("MATTERMOST_SRESUPPORT_NOTIFICATION_HOOK"), payload)
	if err != nil {
		return errors.Wrap(err, "failed tο send Mattermost request payload")
	}
	return nil
}
