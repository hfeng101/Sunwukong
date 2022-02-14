package consts

const(
	HoumaoPhaseCreated = "created"
	HoumaoPhaseXianqi = "xianqi"
	HoumaoPhaseDestroy = "destroying"

	HoumaoOperatorName = "sunwukong"
	HoumaoOperatorNamespace = "huaguoshan"

	HoumaoFilalizer = "houmao.sunwukong"

	ShifaCmdRole = "shifa"
	ZaohuaCmdRole = "zaohua"

	XianqiImage = ""

	XianqiPrefix = "Xianqi_"

	//主动触发，即监听监测指标，主动确认是否要开启造化
	ZaohuaModeZhudong = "Zhudong"
	//被动触发，即接受外部指令，发动造化
	ZaohuaModeBeidong = "Beidong"

	CpuInitializationPeriod = 300
	InitialReadinessDelay = 30
	StabilizationWindowSeconds = 300
	ScaleTolerance = 0.1

	LeaderElectionStateOnStartedLeading = "OnStartedLeading"
	LeaderElectionStateOnNewLeader = "OnNewLeader"
	LeaderElectionStateOnStoppedLeading = "OnStoppedLeading"
)
