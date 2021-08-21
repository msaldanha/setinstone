package timeline

type QueryTypesEnum struct {
	QueryLastEventsRequest  string
	QueryLastEventsResponse string
}

var QueryTypes = QueryTypesEnum{
	QueryLastEventsRequest:  "TIMELINE.QUERY.LASTEVENTS.REQUEST",
	QueryLastEventsResponse: "TIMELINE.QUERY.LASTEVENTS.RESPONSE",
}
