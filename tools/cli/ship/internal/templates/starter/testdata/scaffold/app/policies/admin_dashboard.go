package policies

type PolicyActor struct {
	Email   string
	IsAdmin bool
}

func AdminDashboardAllows(actor PolicyActor) bool {
	return actor.IsAdmin
}
