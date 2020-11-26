package dist

// InteractorState holds all information the interactor needs
type InteractorState struct {
}

// InteractorReq is the request type for the interactor function
type InteractorReq struct {
}

// InteractorRes is the result type for the interactor function
type InteractorRes struct {
}

func (is *InteractorState) interactor(req InteractorReq, res *InteractorRes) (err error) {
	return nil
}
