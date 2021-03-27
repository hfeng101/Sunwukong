package logger

import (
	"github.com/cihub/seelog"
	"log"
)

const (
	LOG_LEVEL_DEBUG = "debug"
	LOG_LEVEL_INFO = "info"
	LOG_LEVEL_WARN = "warn"
	LOG_LEVEL_ERROR = "error"
)

func init() {
	config :=  `
<seelog type="asynctimer" asyncinterval="1000000" minlevel="debug" maxlevel="error">
	<outputs formatid="kube">
		<console/>
		<rollingfile formatid="kube" type="size" filename="niwo.log" maxsize="1000000" maxrolls="5" />
	</outputs>
	<formats>
		<format id="kube" format="[%Date %Time %LEVEL %FullPath %Func %Line]: %Msg%n"/>
	</formats>
</seelog>`
	Logger,err := seelog.LoggerFromConfigAsBytes([]byte(config))
	if err != nil {
		log.Fatal(err)
		return
	}
	seelog.ReplaceLogger(Logger)
}

func SwitchLoggerLevel(level string) {
	config := `
<seelog type="asynctimer" asyncinterval="1000000" minlevel="`+level+`" maxlevel="error">
	<outputs formatid="kube">
		<console/>
		<rollingfile formatid="kube" type="size" filename="niwo.log" maxsize="1000000" maxrolls="5" />
	</outputs>
	<formats>
		<format id="kube" format="[%Date %Time %LEVEL %FullPath %Func %Line]: %Msg%n"/>
	</formats>
</seelog>`

	Logger,err := seelog.LoggerFromConfigAsBytes([]byte(config))
	if err != nil {
		log.Fatal(err)
		return
	}

	seelog.ReplaceLogger(Logger)
}


