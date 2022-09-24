package model

type FunctionsBlock struct {
	Marker    string
	Functions []*Function
}

type Function struct {
	Comments    []string
	Name        string
	Receiver    string
	Src         Var
	Dst         Var
	RetError    bool
	DstVarStyle DstVarStyle
	Assignments []*Assignment
	PreProcess  *Manipulator
	PostProcess *Manipulator
}
