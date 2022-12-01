package main

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	mmodel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
	"github.com/opsgenie/opsgenie-go-sdk-v2/user"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Starting oncall notifier...")

	err := checkEnvVariables()
	if err != nil {
		log.WithError(err).Error("Failed to pass env variable checks")
	}

	err = handleGroups()
	if err != nil {
		log.WithError(err).Error("Failed to run handleOnCall function")
	}
}

func checkEnvVariables() error {
	var envVariables = []string{
		"OPSGENIE_APIKEY",
		"MATTERMOST_SREONCALL_NOTIFICATION_HOOK",
		"MATTERMOST_SRESUPPORT_NOTIFICATION_HOOK",
		"MATTERMOST_BOT_TOKEN",
		"MATTERMOST_URL",
		"MATTERMOST_SRESUPPORT_GROUPID",
		"MATTERMOST_SREONCALL_GROUPID",
		"ONCALL_HOUR_SHIFTS",
		"SUPPORT_APPROVED_LIST",
		"SUPPORT_OVERRIDE_LIST",
		"PRIMARY_ONCALL_GROUP_NAME",
		"SECONDARY_ONCALL_GROUP_NAME",
	}

	for _, envVar := range envVariables {
		if os.Getenv(envVar) == "" {
			return errors.Errorf("Environment variable %s was not set", envVar)
		}
	}
	return nil
}

func whoIsOnCall(userNameType string) (string, string, string, string, error) {

	primaryNow, err := getOncall(os.Getenv("PRIMARY_ONCALL_GROUP_NAME"), userNameType, time.Now().UTC())
	if err != nil || primaryNow == "" {
		return "", "", "", "", errors.Wrap(err, "not able to get who is the primary now")
	}

	secondaryNow, err := getOncall(os.Getenv("SECONDARY_ONCALL_GROUP_NAME"), userNameType, time.Now().UTC())
	if err != nil || secondaryNow == "" {
		return "", "", "", "", errors.Wrap(err, "not able to get who is the secondary now")
	}

	intVar, err := strconv.Atoi(os.Getenv("ONCALL_HOUR_SHIFTS"))
	if err != nil {
		return "", "", "", "", errors.Wrap(err, "not able to get integer from oncall hour shifts")
	}

	primaryLater, err := getOncall(os.Getenv("PRIMARY_ONCALL_GROUP_NAME"), userNameType, time.Now().UTC().Add(time.Hour*time.Duration(intVar)))
	if err != nil || primaryLater == "" {
		return "", "", "", "", errors.Wrap(err, "not able to get who is the primary later")
	}

	secondaryLater, err := getOncall(os.Getenv("SECONDARY_ONCALL_GROUP_NAME"), userNameType, time.Now().UTC().Add(time.Hour*time.Duration(intVar)))
	if err != nil || secondaryLater == "" {
		log.Fatal("not able to get who is the secondary later")
		return "", "", "", "", errors.Wrap(err, "not able to get who is the secondary later")
	}

	return primaryNow, secondaryNow, primaryLater, secondaryLater, nil
}

func handleGroups() error {
	mmClient := mmodel.NewAPIv4Client("https://community.mattermost.com")
	mmClient.SetOAuthToken(os.Getenv("MATTERMOST_BOT_TOKEN"))

	primaryNow, secondaryNow, primaryLater, secondaryLater, err := whoIsOnCall("mattermost_username")
	if err != nil {
		return err
	}

	log.Infof("The Primary oncall right now is %s", primaryNow)
	log.Infof("The Secondary oncall right now is %s", secondaryNow)
	log.Infof("The Primary oncall later is %s", primaryLater)
	log.Infof("The Secondary oncall later is %s", secondaryLater)

	primaryNowUserID, err := getMMUserID(primaryNow, mmClient)
	if err != nil {
		return err
	}

	secondaryNowUserID, err := getMMUserID(secondaryNow, mmClient)
	if err != nil {
		return err
	}

	supportUsers := getSupportFromApprovedList([]string{secondaryNow, secondaryLater})
	var supportUserIDs []string
	for _, user := range supportUsers {
		userID, err := getMMUserID(user, mmClient)
		if err != nil {
			return err
		}
		supportUserIDs = append(supportUserIDs, userID)
	}

	log.Info("Getting existing @sreoncall group members")
	onCallGroupMembers, _ := getMMGroupUsers(os.Getenv("MATTERMOST_SREONCALL_GROUPID"), mmClient)
	log.Infof("Currently SRE OnCall in Mattermost Group is:")
	var onCallMemberIDs []string
	for _, member := range onCallGroupMembers {
		log.Infof("%s.%s", strings.ToLower(member.FirstName), strings.ToLower(member.LastName))
		onCallMemberIDs = append(onCallMemberIDs, member.Id)
	}

	if !checkEquality(onCallMemberIDs, []string{primaryNowUserID, secondaryNowUserID}) {
		log.Info("Cleaning existing @sreoncall group members")
		err = removeMMGroupUsers(os.Getenv("MATTERMOST_SREONCALL_GROUPID"), onCallMemberIDs, mmClient)
		if err != nil {
			return err
		}

		log.Infof("Setting %s as primary and %s as secondary in @sreoncall group", primaryNow, secondaryNow)
		err = setMMGroupUsers(os.Getenv("MATTERMOST_SREONCALL_GROUPID"), []string{primaryNowUserID, secondaryNowUserID}, mmClient)
		if err != nil {
			return err
		}

		err = sendWhoIsOnCallNotification(primaryNow, secondaryNow)
		if err != nil {
			return err
		}
	} else {
		log.Info("@sreoncall group is already synced")
	}

	log.Info("Getting existing @sresupport group members")
	sreSupportGroupMembers, _ := getMMGroupUsers(os.Getenv("MATTERMOST_SRESUPPORT_GROUPID"), mmClient)
	log.Infof("Currently SRE Support in Mattermost Group is:")
	var supportMemberIDs []string
	for _, member := range sreSupportGroupMembers {
		log.Infof("%s.%s", strings.ToLower(member.FirstName), strings.ToLower(member.LastName))
		supportMemberIDs = append(supportMemberIDs, member.Id)
	}

	if !checkEquality(supportMemberIDs, supportUserIDs) {
		log.Info("Cleaning existing @sresupport group members")
		err = removeMMGroupUsers(os.Getenv("MATTERMOST_SRESUPPORT_GROUPID"), supportMemberIDs, mmClient)
		if err != nil {
			return err
		}
		log.Infof("Setting %s and %s as sresupport in @sresupport group", supportUsers[0], supportUsers[1])
		err = setMMGroupUsers(os.Getenv("MATTERMOST_SRESUPPORT_GROUPID"), supportUserIDs, mmClient)
		if err != nil {
			return err
		}

		err = sendWhoIsSRESupportNotification(supportUsers[0], supportUsers[1])
		if err != nil {
			return err
		}
	} else {
		log.Info("@sresupport group is already synced")
	}

	return nil
}

func getSupportFromApprovedList(members []string) []string {
	var supportMembers []string
	approvedList := strings.Split(os.Getenv("SUPPORT_APPROVED_LIST"), ",")
	for _, member := range members {
		approved := false
		for _, approvedMember := range approvedList {
			if member == approvedMember {
				supportMembers = append(supportMembers, member)
				approved = true
			}
		}
		if !approved {
			supportOverrideList := strings.Split(os.Getenv("SUPPORT_OVERRIDE_LIST"), ",")
			rand.Seed(time.Now().Unix())
			n := rand.Int() % len(supportOverrideList)
			supportMembers = append(supportMembers, supportOverrideList[n])
		}
	}
	return supportMembers
}

func setMMGroupUsers(groupID string, userIDs []string, mmClient *mmodel.Client4) error {
	members := &mmodel.GroupModifyMembers{UserIds: userIDs}
	_, _, err := mmClient.UpsertGroupMembers(groupID, members)
	if err != nil {
		return errors.Wrap(err, "unable to set Mattermost Group members")
	}
	return nil
}

func removeMMGroupUsers(groupID string, userIDs []string, mmClient *mmodel.Client4) error {
	members := &mmodel.GroupModifyMembers{UserIds: userIDs}
	_, _, err := mmClient.DeleteGroupMembers(groupID, members)
	if err != nil {
		return errors.Wrap(err, "unable to remove Mattermost Group members")
	}
	return nil
}

func getMMUserID(userName string, mmClient *mmodel.Client4) (string, error) {
	user, _, err := mmClient.GetUserByUsername(userName, "")
	if err != nil {
		return "", errors.Wrap(err, "unable to get Mattermost user by username")
	}
	return user.Id, nil
}

func getMMGroupUsers(groupID string, mmClient *mmodel.Client4) ([]*mmodel.User, error) {
	users, _, err := mmClient.GetUsersInGroup(groupID, 0, 100, "")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get Mattermost users")
	}
	return users, nil
}

func getUserInfo(opsGenieUser, userNameType string) string {
	client, err := user.NewClient(&client.Config{
		ApiKey: os.Getenv("OPSGENIE_APIKEY"),
	})
	if err != nil {
		log.Fatal("not able to create a new opsgenie client")
		return ""
	}

	userReq := &user.GetRequest{
		Identifier: opsGenieUser,
	}

	userResult, err := client.Get(context.Background(), userReq)
	if err != nil {
		log.Fatal("not able to get the user")
		return ""
	}

	log.Println(userResult.FullName, userResult.Username)
	if len(userResult.Details[userNameType]) == 0 {
		return userResult.FullName
	}

	return userResult.Details[userNameType][0]
}

func getOncall(scheduleName, userNameType string, date time.Time) (string, error) {
	client, err := schedule.NewClient(&client.Config{
		ApiKey:   os.Getenv("OPSGENIE_APIKEY"),
		LogLevel: log.DebugLevel,
	})
	if err != nil {
		log.Fatal("not able to create a new opsgenie client")
		return "", err
	}

	flat := true
	onCallReq := &schedule.GetOnCallsRequest{
		Flat:                   &flat,
		Date:                   &date,
		ScheduleIdentifierType: schedule.Name,
		ScheduleIdentifier:     scheduleName,
	}
	onCall, err := client.GetOnCalls(context.TODO(), onCallReq)
	if err != nil {
		log.Fatal("not able to get who is on call 2")
		return "", err
	}

	if (len(onCall.OnCallRecipients)) <= 0 {
		return "", nil
	}

	primary := getUserInfo(onCall.OnCallRecipients[0], userNameType)
	if primary == "" {
		return onCall.OnCallRecipients[0], nil
	}

	return primary, nil
}

func checkEquality(slice1, slice2 []string) bool {
	var found int
	if len(slice1) != len(slice2) {
		return false
	}
	for _, el1 := range slice1 {
		for _, el2 := range slice2 {
			if el1 == el2 {
				found++
				break
			}
		}
	}
	return found == len(slice1)
}
