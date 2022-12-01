package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	model "github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func send(webhookURL string, payload model.CommandResponse) error {
	marshalContent, _ := json.Marshal(payload)
	var jsonStr = []byte(marshalContent)
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "aws-sns")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed tο send HTTP request")
	}
	defer resp.Body.Close()
	return nil
}

func sendWhoIsOnCallNotification(primary, secondary string) error {
	attachment := &model.SlackAttachment{
		Title:     "SRE Oncall",
		TitleLink: "https://mattermost.app.opsgenie.com/alert",
		Color:     "#0000ff",
		Fields: []*model.SlackAttachmentField{
			{Title: "Who is onCall?", Short: false},
			{Title: "Primary", Value: fmt.Sprintf("@%s", primary), Short: true},
			{Title: "Secondary", Value: fmt.Sprintf("@%s", secondary), Short: true},
			{Value: "_Who you gonna call?_ Use: @sreoncall", Short: false},
		},
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

func sendWhoIsSRESupportNotification(user1, user2 string) error {
	attachment := &model.SlackAttachment{
		Title:     "SRE Support",
		TitleLink: "https://mattermost.app.opsgenie.com/alert",
		Color:     "#0000ff",
		Fields: []*model.SlackAttachmentField{
			{Title: "Who is on SRE Support?", Short: false},
			{Title: "Support A", Value: fmt.Sprintf("@%s", user1), Short: true},
			{Title: "Support B", Value: fmt.Sprintf("@%s", user2), Short: true},
			{Value: "_Who you gonna ask for help?_ Use: @sresupport", Short: false},
		},
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
