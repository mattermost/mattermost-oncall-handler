package main

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	mmodel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
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
		"PAGERDUTY_APIKEY",
		"MATTERMOST_SREONCALL_NOTIFICATION_HOOK",
		"MATTERMOST_SRESUPPORT_NOTIFICATION_HOOK",
		"MATTERMOST_BOT_TOKEN",
		"MATTERMOST_URL",
		"MATTERMOST_SRESUPPORT_GROUPID",
		"MATTERMOST_SREONCALL_GROUPID",
		"ONCALL_HOUR_SHIFTS",
		"SUPPORT_APPROVED_LIST",
		"SUPPORT_OVERRIDE_LIST",
		"PRIMARY_SCHEDULE_ID",
		"SECONDARY_SCHEDULE_ID",
	}

	for _, envVar := range envVariables {
		if os.Getenv(envVar) == "" {
			return errors.Errorf("Environment variable %s was not set", envVar)
		}
	}
	return nil
}

func whoIsOnCall(userNameType string) (string, string, string, string, error) {

	client := pagerduty.NewClient(os.Getenv("PAGERDUTY_APIKEY"))

	intVar, err := strconv.Atoi(os.Getenv("ONCALL_HOUR_SHIFTS"))
	if err != nil {
		return "", "", "", "", errors.Wrap(err, "not able to get integer from oncall hour shifts")
	}

	now := time.Now().UTC()
	future := time.Now().UTC().Add(time.Hour * time.Duration(intVar))

	// Set up options to get who is on call now for a specific team
	optsNowPrimary := pagerduty.ListOnCallOptions{
		Since:       now.Format(time.RFC3339),
		Until:       now.Format(time.RFC3339),
		ScheduleIDs: []string{os.Getenv("PRIMARY_SCHEDULE_ID")},
	}

	// Set up options to get who will be on call later for the specific team
	optsFuturePrimary := pagerduty.ListOnCallOptions{
		Since:       future.Format(time.RFC3339),
		Until:       future.Format(time.RFC3339),
		ScheduleIDs: []string{os.Getenv("PRIMARY_SCHEDULE_ID")},
	}

	// Set up options to get who is on call now for a specific team
	optsNowSecondary := pagerduty.ListOnCallOptions{
		Since:       now.Format(time.RFC3339),
		Until:       now.Format(time.RFC3339),
		ScheduleIDs: []string{os.Getenv("SECONDARY_SCHEDULE_ID")},
	}

	// Set up options to get who will be on call later for the specific team
	optsFutureSecondary := pagerduty.ListOnCallOptions{
		Since:       future.Format(time.RFC3339),
		Until:       future.Format(time.RFC3339),
		ScheduleIDs: []string{os.Getenv("SECONDARY_SCHEDULE_ID")},
	}

	onCallsNowPrimary, err := client.ListOnCallsWithContext(context.TODO(), optsNowPrimary)
	if err != nil {
		log.WithError(err).Error("Error fetching current on-call information for the primary team")
		return "", "", "", "", err
	}

	userNowPrimary, err := client.GetUserWithContext(context.TODO(), onCallsNowPrimary.OnCalls[0].User.ID, pagerduty.GetUserOptions{})
	if err != nil {
		log.WithError(err).Error("Error fetching user details for primary oncall now")
		return "", "", "", "", err
	}
	log.Infof("User Now Primary: %s (%s)\n", userNowPrimary.Name, userNowPrimary.Email)
	usernameNowPrimary := strings.ReplaceAll(strings.ToLower(userNowPrimary.Name), " ", ".")

	onCallsNowSecondary, err := client.ListOnCallsWithContext(context.TODO(), optsNowSecondary)
	if err != nil {
		log.WithError(err).Error("Error fetching current on-call information for the secondary team")
		return "", "", "", "", err
	}

	userNowSecondary, err := client.GetUserWithContext(context.TODO(), onCallsNowSecondary.OnCalls[0].User.ID, pagerduty.GetUserOptions{})
	if err != nil {
		log.WithError(err).Error("Error fetching user details for secondary oncall now")
		return "", "", "", "", err
	}
	log.Infof("User Now Secondary: %s (%s)\n", userNowSecondary.Name, userNowSecondary.Email)
	usernameNowSecondary := strings.ReplaceAll(strings.ToLower(userNowSecondary.Name), " ", ".")

	onCallsFuturePrimary, err := client.ListOnCallsWithContext(context.TODO(), optsFuturePrimary)
	if err != nil {
		log.WithError(err).Error("Error fetching future on-call information for the primary team")
		return "", "", "", "", err
	}

	userFuturePrimary, err := client.GetUserWithContext(context.TODO(), onCallsFuturePrimary.OnCalls[0].User.ID, pagerduty.GetUserOptions{})
	if err != nil {
		log.WithError(err).Error("Error fetching user details for primary oncall in the future")
		return "", "", "", "", err
	}
	log.Infof("User Future Primary: %s (%s)\n", userFuturePrimary.Name, userFuturePrimary.Email)
	usernameFuturePrimary := strings.ReplaceAll(strings.ToLower(userFuturePrimary.Name), " ", ".")

	onCallsFutureSecondary, err := client.ListOnCallsWithContext(context.TODO(), optsFutureSecondary)
	if err != nil {
		log.WithError(err).Error("Error fetching future on-call information for the secondary team")
		return "", "", "", "", err
	}

	userFutureSecondary, err := client.GetUserWithContext(context.TODO(), onCallsFutureSecondary.OnCalls[0].User.ID, pagerduty.GetUserOptions{})
	if err != nil {
		log.WithError(err).Error("Error fetching user details for secondary oncall in the future")
		return "", "", "", "", err
	}
	log.Infof("User Future Secondary: %s (%s)\n", userFutureSecondary.Name, userFutureSecondary.Email)
	usernameFutureSecondary := strings.ReplaceAll(strings.ToLower(userFutureSecondary.Name), " ", ".")

	return usernameNowPrimary, usernameNowSecondary, usernameFuturePrimary, usernameFutureSecondary, nil
}

func handleGroups() error {
	mmClient := mmodel.NewAPIv4Client(os.Getenv("MATTERMOST_URL"))
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
		if primaryNow == secondaryNow {
			err = setMMGroupUsers(os.Getenv("MATTERMOST_SREONCALL_GROUPID"), []string{primaryNowUserID}, mmClient)
			if err != nil {
				return err
			}
		} else {
			err = setMMGroupUsers(os.Getenv("MATTERMOST_SREONCALL_GROUPID"), []string{primaryNowUserID, secondaryNowUserID}, mmClient)
			if err != nil {
				return err
			}
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
	if len(sreSupportGroupMembers) > 0 {
		for _, member := range sreSupportGroupMembers {
			log.Infof("%s.%s", strings.ToLower(member.FirstName), strings.ToLower(member.LastName))
			supportMemberIDs = append(supportMemberIDs, member.Id)
		}
	}

	if !checkEquality(supportMemberIDs, supportUserIDs) {
		if len(sreSupportGroupMembers) > 0 {
			log.Info("Cleaning existing @sresupport group members")
			err = removeMMGroupUsers(os.Getenv("MATTERMOST_SRESUPPORT_GROUPID"), supportMemberIDs, mmClient)
			if err != nil {
				return err
			}
		}
		log.Infof("Setting %s as sresupport in @sresupport group", strings.Join(supportUsers, ","))
		err = setMMGroupUsers(os.Getenv("MATTERMOST_SRESUPPORT_GROUPID"), supportUserIDs, mmClient)
		if err != nil {
			return err
		}

		err = sendWhoIsSRESupportNotification(supportUsers)
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
	var approved bool
	for _, member := range members {
		for _, approvedMember := range approvedList {
			if member == approvedMember {
				supportMembers = append(supportMembers, member)
				approved = true
			}
		}

	}
	if !approved {
		if os.Getenv("SUPPORT_OVERRIDE_LIST") != "" {
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
