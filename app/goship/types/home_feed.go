package types

type (
	HomeFeedData struct {
		NextPageURL           string
		RanOutOfQuestions     bool
		SupportEmail          string
		NumFriends            int
		JustFinishedOnboarded bool
	}

	HomeFeedButtonsData struct {
		NumDrafts           int
		NumLikedQuestions   int
		NumWaitingOnPartner int
		NumWaitingOnYou     int
		MaxNumCanWaitOnYou  int
	}

	HomeFeedStatsData struct {
		NumAnsweredInLast24H int
	}
)
