package mk2rbc

import (
	"fmt"
	"io"

	"go.starlark.net/syntax"
)

func emit(stmts []syntax.Stmt, w io.Writer) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *syntax.AssignStmt:
			return emitAssign(s, w)
		case *syntax.BranchStmt:
			return emitBranch(s, w)
		case *syntax.DefStmt:
			return emitDef(s, w)
		case *syntax.ExprStmt:
			if err := emitExpr(s.X, w); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		case *syntax.ForStmt:
			return emitFor(s, w)
		case *syntax.WhileStmt:
			return emitWhile(s, w)
		case *syntax.IfStmt:
			return emitIf(s, w, false)
		case *syntax.LoadStmt:
			return emitLoad(s, w)
		case *syntax.ReturnStmt:
			return emitReturn(s, w)
		default:
			return fmt.Errorf("unknown starlark statement type")
		}
	}

	return nil
}

func emitAssign(s *syntax.AssignStmt, w io.Writer) error {
	if err := emitExpr(s.LHS, w); err != nil {
		return err
	}

	op := ""
	switch s.Op {
	case syntax.PLUS:
		op = " + "
	case syntax.PLUS_EQ:
		op = " += "
	case syntax.MINUS_EQ:
		op = " -= "
	case syntax.STAR_EQ:
		op = " *= "
	case syntax.PERCENT_EQ:
		op = " %= "
	default:
		return fmt.Errorf("unknown op for assignment: %d", s.Op)
	}
	if _, err := fmt.Fprint(w, op); err != nil {
		return err
	}

	if err := emitExpr(s.RHS, w); err != nil {
		return err
	}

	return nil
}

func emitBranch(s *syntax.BranchStmt, w io.Writer) error {
	switch s.Token {
	case syntax.BREAK:
		if _, err := fmt.Fprintln(w, "break"); err != nil {
			return err
		}
	case syntax.CONTINUE:
		if _, err := fmt.Fprintln(w, "continue"); err != nil {
			return err
		}
	case syntax.PASS:
		if _, err := fmt.Fprintln(w, "pass"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown token for branch: %d", s.Token)
	}
	return nil
}

func emitDef(s *syntax.DefStmt, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "def %s(", s.Name.Name); err != nil {
		return err
	}

	if err := emitCommaSeparatedExprList(s.Params, w); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "):"); err != nil {
		return err
	}

	if err := emit(s.Body, w); err != nil {
		return err
	}

	return nil
}

func emitFor(s *syntax.ForStmt, w io.Writer) error {
	if _, err := fmt.Fprint(w, "for "); err != nil {
		return err
	}

	if err := emitExpr(s.Vars, w); err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, " in "); err != nil {
		return err
	}

	if err := emitExpr(s.X, w); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, ":"); err != nil {
		return err
	}

	if err := emit(s.Body, w); err != nil {
		return err
	}

	return nil
}

func emitWhile(s *syntax.WhileStmt, w io.Writer) error {
	if _, err := fmt.Fprint(w, "while "); err != nil {
		return err
	}

	if err := emitExpr(s.Cond, w); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, ":"); err != nil {
		return err
	}

	if err := emit(s.Body, w); err != nil {
		return err
	}

	return nil
}

func emitIf(s *syntax.IfStmt, w io.Writer, isElif bool) error {
	if isElif {
		if _, err := fmt.Fprint(w, "elseif "); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprint(w, "if "); err != nil {
			return err
		}
	}

	if err := emitExpr(s.Cond, w); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, ":"); err != nil {
		return err
	}

	if err := emit(s.True, w); err != nil {
		return err
	}

	if len(s.False) == 1 {
		if ifStmt, ok := s.False[0].(*syntax.IfStmt); ok {
			return emitIf(ifStmt, w, true)
		}
	}

	if len(s.False) > 0 {
		if _, err := fmt.Fprintln(w, "else:"); err != nil {
			return err
		}
		return emit(s.False, w)
	}

	return nil
}


func emitLoad(s *syntax.LoadStmt, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "load(%q", s.Module.Raw); err != nil {
		return err
	}

	for i, from := range s.From {
		if _, err := fmt.Fprintf(w, ", %s=%q", s.To[i].Name, from.Name); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w, ")")
	return err
}

func emitReturn(s *syntax.ReturnStmt, w io.Writer) error {
	if s.Result == nil {
		_, err := fmt.Fprintln(w, "return")
		return err
	}

	if _, err := fmt.Fprintf(w, "return "); err != nil {
		return err
	}

	if err := emitExpr(s.Result, w); err != nil {
		return err
	}

	_, err := fmt.Fprintln(w)
	return err
}

func emitExpr(expr syntax.Expr, w io.Writer) error {
	switch e := expr.(type) {
	case *syntax.BinaryExpr:
		return emitBinaryExpr(e, w)
	case *syntax.CallExpr:
		return emitCallExpr(e, w)
	case *syntax.Comprehension:
		return emitComprehension(e, w)
	case *syntax.CondExpr:
		return emitCondExpr(e, w)
	case *syntax.DictEntry:
		return emitDictEntry(e, w)
	case *syntax.DictExpr:
		return emitDictExpr(e, w)
	case *syntax.DotExpr:
		return emitDotExpr(e, w)
	case *syntax.Ident:
		return emitIdent(e, w)
	case *syntax.IndexExpr:
		return emitIndexExpr(e, w)
	case *syntax.LambdaExpr:
		return emitLambdaExpr(e, w)
	case *syntax.ListExpr:
		return emitListExpr(e, w)
	case *syntax.Literal:
		return emitLiteralExpr(e, w)
	case *syntax.ParenExpr:
		return emitParenExpr(e, w)
	case *syntax.SliceExpr:
		return emitSliceExpr(e, w)
	case *syntax.TupleExpr:
		return emitTupleExpr(e, w)
	case *syntax.UnaryExpr:
		return emitUnaryExpr(e, w)
	default:
		return fmt.Errorf("unknown starlark expression type")
	}
}

func emitBinaryExpr(e *syntax.BinaryExpr, w io.Writer) error {
	if err := emitExpr(e.X, w); err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, e.Op.String()); err != nil {
		return err
	}

	return emitExpr(e.Y, w)
}

func emitCallExpr(e *syntax.CallExpr, w io.Writer) error {
	if err := emitExpr(e.Fn, w); err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, "("); err != nil {
		return err
	}

	if err := emitCommaSeparatedExprList(e.Args, w); err != nil {
		return err
	}

	_, err := fmt.Fprint(w, ")")
	return err
}

func emitComprehension(e *syntax.Comprehension, w io.Writer) error {
	if e.Curly {
		if _, err := fmt.Fprint(w, "{"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprint(w, "["); err != nil {
			return err
		}
	}

	if err := emitExpr(e.Body, w); err != nil {
		return err
	}

	for _, clause := range e.Clauses {
		switch clause := clause.(type) {
		case *syntax.ForClause:
			if _, err := fmt.Fprint(w, " for "); err != nil {
				return err
			}
			if err := emitExpr(clause.Vars, w); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, " in "); err != nil {
				return err
			}
			if err := emitExpr(clause.X, w); err != nil {
				return err
			}
		case *syntax.IfClause:
			if _, err := fmt.Fprint(w, " if "); err != nil {
				return err
			}
			if err := emitExpr(clause.Cond, w); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown clause type")
		}
	}

	if e.Curly {
		_, err := fmt.Fprint(w, "}")
		return err
	} else {
		_, err := fmt.Fprint(w, "]")
		return err
	}
}

func emitCondExpr(e *syntax.CondExpr, w io.Writer) error {
	if err := emitExpr(e.True, w); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, " if "); err != nil {
		return err
	}
	if err := emitExpr(e.Cond, w); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, " else "); err != nil {
		return err
	}
	return emitExpr(e.False, w)
}

func emitDictEntry(e *syntax.DictEntry, w io.Writer) error {
	if err := emitExpr(e.Key, w); err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, ": "); err != nil {
		return err
	}

	return emitExpr(e.Value, w)
}

func emitDictExpr(e *syntax.DictExpr, w io.Writer) error {
	if _, err := fmt.Fprint(w, "{"); err != nil {
		return err
	}

	if err := emitCommaSeparatedExprList(e.List, w); err != nil {
		return err
	}

	_, err := fmt.Fprint(w, "}")
	return err
}

func emitDotExpr(e *syntax.DotExpr, w io.Writer) error {
	if err := emitExpr(e.X, w); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, ".%s", e.Name.Name)
	return err
}

func emitIdent(e *syntax.Ident, w io.Writer) error {
	_, err := fmt.Fprint(w, e.Name)
	return err
}

func emitIndexExpr(e *syntax.IndexExpr, w io.Writer) error {
	if err := emitExpr(e.X, w); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "["); err != nil {
		return err
	}
	if err := emitExpr(e.Y, w); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, "]")
	return err
}

func emitLambdaExpr(e *syntax.LambdaExpr, w io.Writer) error {
	if _, err := fmt.Fprint(w, "lambda "); err != nil {
		return err
	}
	if err := emitCommaSeparatedExprList(e.Params, w); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, ": "); err != nil {
		return err
	}
	return emitExpr(e.Body, w)
}

func emitListExpr(e *syntax.ListExpr, w io.Writer) error {
	if _, err := fmt.Fprint(w, "["); err != nil {
		return err
	}
	if err := emitCommaSeparatedExprList(e.List, w); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, "]")
	return err
}

func emitLiteralExpr(e *syntax.Literal, w io.Writer) error {
	_, err := fmt.Fprint(w, e.Raw)
	return err
}

func emitParenExpr(e *syntax.ParenExpr, w io.Writer) error {
	if _, err := fmt.Fprint(w, "("); err != nil {
		return err
	}
	if err := emitExpr(e.X, w); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, ")")
	return err
}

func emitSliceExpr(e *syntax.SliceExpr, w io.Writer) error {
	if err := emitExpr(e.X, w); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "["); err != nil {
		return err
	}
	sep := ""
	if e.Lo != nil {
		if err := emitExpr(e.Lo, w); err != nil {
			return err
		}
		sep = ":"
	}
	if e.Hi != nil {
		if _, err := fmt.Fprint(w, sep); err != nil {
			return err
		}
		if err := emitExpr(e.Hi, w); err != nil {
			return err
		}
		sep = ":"
	}
	if e.Step != nil {
		if _, err := fmt.Fprint(w, sep); err != nil {
			return err
		}
		if err := emitExpr(e.Step, w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "]")
	return err
}

func emitTupleExpr(e *syntax.TupleExpr, w io.Writer) error {
	if e.Lparen.Line != 0 {
		if _, err := fmt.Fprint(w, "("); err != nil {
			return err
		}
	}
	if err := emitCommaSeparatedExprList(e.List, w); err != nil {
		return err
	}
	if e.Lparen.Line != 0 {
		if _, err := fmt.Fprint(w, ")"); err != nil {
			return err
		}
	}
	return nil
}

func emitUnaryExpr(e *syntax.UnaryExpr, w io.Writer) error {
	if _, err := fmt.Fprint(w, e.Op.String()); err != nil {
		return err
	}
	if e.X == nil && e.Op != syntax.STAR {
		return fmt.Errorf("UnaryExpr's operand must be non-nil if the operator is not a *")
	}
	if e.X != nil {
		return emitExpr(e.X, w)
	}
	return nil
}

func emitCommaSeparatedExprList(exprs []syntax.Expr, w io.Writer) error {
	for i, e := range exprs {
		if err := emitExpr(e, w); err != nil {
			return err
		}

		if i < len(exprs) - 1 {
			if _, err := fmt.Fprint(w, ", "); err != nil {
				return err
			}
		}
	}
	return nil
}
