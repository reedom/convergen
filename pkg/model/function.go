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
	PreProcess   Manipulator
	PostProcess  Manipulator
}

type Manipulator struct {
	Name  string
	Error bool
}
