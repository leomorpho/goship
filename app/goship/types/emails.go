package types

type (
	EmailDefaultData struct {
		AppName          string
		SupportEmail     string
		Domain           string
		ConfirmationLink string
	}

	EmailPasswordResetData struct {
		AppName           string
		SupportEmail      string
		Domain            string
		ProfileName       string
		PasswordResetLink string
		OperatingSystem   string
		BrowserName       string
	}

	QuestionInEmail struct {
		Question       string
		WriteAnswerURL string
	}

	EmailUpdate struct {
		SelfName                                 string
		AppName                                  string
		SupportEmail                             string
		Domain                                   string
		PartnerName                              string
		NumNewNotifications                      int
		QuestionsAnsweredByFriendButNotSelfTitle string
		NumQuestionsAnsweredByFriendButNotSelf   int
		QuestionsAnsweredByFriendButNotSelf      []QuestionInEmail
		QuestionsNotAnsweredInSocialCircle       []QuestionInEmail
		UnsubscribeDailyUpdatesLink              string
		UnsubscribePartnerActivityLink           string
	}
)
