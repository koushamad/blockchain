package Handler

import "log"

func Handle(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
