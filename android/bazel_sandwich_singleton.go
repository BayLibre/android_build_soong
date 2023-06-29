package android

func bazelSandwichSingletonFactory() Singleton {
	return &bazelSandwichSingleton{}
}

type bazelSandwichSingleton struct {
}

func (bazelSandwichSingleton) QueueBazelCall(ctx BaseModuleContext) {

}

func (bazelSandwichSingleton) GenerateBuildActions(ctx SingletonContext) {

}
