package model

type Function struct {
	Comments     []string
	Name         string
	Receiver     string
	Src          Var
	Dst          Var
	ReturnsError bool
	DstVarStyle  DstVarStyle
	Assignments  []*Assignment
	PreProcess   *Manipulator
	PostProcess  *Manipulator
}

type Manipulator struct {
	Pkg          string
	Name         string
	Dst          Var
	Src          Var
	ReturnsError bool
}
