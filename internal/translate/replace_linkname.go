package translate

var replaceLinkname = map[packageSelector]packageSelector{
	{Pkg: "runtime", Selector: "cheaprandn"}:     {Pkg: gosimruntimePackage, Selector: "fastrandn"},
	{Pkg: "runtime", Selector: "findfunc"}:       {Pkg: gosimruntimePackage, Selector: "findfunc"},
	{Pkg: "runtime", Selector: "callers"}:        {Pkg: gosimruntimePackage, Selector: "callers"},
	{Pkg: "runtime", Selector: "setCrashFD"}:     {Pkg: gosimruntimePackage, Selector: "setCrashFD"},
	{Pkg: "runtime", Selector: "funcInfo.entry"}: {Pkg: gosimruntimePackage, Selector: "funcInfo.entry"},
}
