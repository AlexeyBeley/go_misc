package list

import (
	"github.com/AlexeyBeley/go_common/logger"
)

//var lg = &(logger.Logger{Level: logger.DEBUG})
var lg = &(logger.Logger{Level: logger.INFO})

type List interface{
	Insert (any)
	Print ()
}

