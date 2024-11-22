package translate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func isRecvExpr(node dst.Node) (*dst.UnaryExpr, bool) {
	expr, ok := node.(dst.Expr)
	if !ok {
		return nil, false
	}
	expr = dstutil.Unparen(expr)
	if recvExpr, ok := expr.(*dst.UnaryExpr); ok && recvExpr.Op == token.ARROW {
		return recvExpr, true
	}
	return nil, false
}

type ChanType struct {
	Type  *types.Chan
	Named types.Type
}

func (t *packageTranslator) isChanType(expr dst.Expr) (ChanType, bool) {
	if astExpr, ok := t.astMap.Nodes[expr].(ast.Expr); ok {
		if convertedType, ok := t.implicitConversions[astExpr]; ok {
			if chanType, ok := convertedType.Underlying().(*types.Chan); ok {
				return ChanType{Type: chanType, Named: convertedType}, true
			}
		}
	}

	if typ, ok := t.getType(expr); ok {
		if chanTyp, ok := typ.Underlying().(*types.Chan); ok {
			return ChanType{Type: chanTyp, Named: typ}, true
		}
	}

	return ChanType{}, false
}

func (t *packageTranslator) maybeConvertChanType(x dst.Expr, typ ChanType) dst.Expr {
	if typ.Named == typ.Type {
		return x
	}

	return &dst.CallExpr{
		Fun: t.makeTypeExpr(typ.Named),
		Args: []dst.Expr{
			x,
		},
	}
}

func (t *packageTranslator) maybeExtractNamedChanType(x dst.Expr, typ ChanType) dst.Expr {
	if typ.Named == typ.Type {
		return x
	}
	return &dst.CallExpr{Fun: t.newRuntimeSelector("ExtractChan"), Args: []dst.Expr{x}}
}

func (t *packageTranslator) rewriteChanLen(c *dstutil.Cursor) {
	if call, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(call.Fun, "len") {
		if typ, ok := t.isChanType(call.Args[0]); ok {
			call.Fun = &dst.SelectorExpr{
				X:   t.maybeExtractNamedChanType(call.Args[0], typ),
				Sel: dst.NewIdent("Len"),
			}
			call.Args = nil
		}
	}
}

func (t *packageTranslator) rewriteChanCap(c *dstutil.Cursor) {
	if call, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(call.Fun, "cap") {
		if typ, ok := t.isChanType(call.Args[0]); ok {
			call.Fun = &dst.SelectorExpr{
				X:   t.maybeExtractNamedChanType(call.Args[0], typ),
				Sel: dst.NewIdent("Cap"),
			}
			call.Args = nil
		}
	}
}

func (t *packageTranslator) rewriteChanType(c *dstutil.Cursor) {
	// chan type
	if chanType, ok := c.Node().(*dst.ChanType); ok {
		// XXX: this ignores the arrow...
		c.Replace(&dst.IndexListExpr{
			X: t.newRuntimeSelector("Chan"),
			Indices: []dst.Expr{
				chanType.Value, // XXX: apply?
			},
		})
	}
}

func (t *packageTranslator) rewriteChanRange(c *dstutil.Cursor) {
	if rangeStmt, ok := c.Node().(*dst.RangeStmt); ok {
		if typ, ok := t.isChanType(rangeStmt.X); ok {
			rangeStmt.X = &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   t.maybeExtractNamedChanType(rangeStmt.X, typ),
					Sel: dst.NewIdent("Range"),
				},
			}
		}
	}
}

func (t *packageTranslator) rewriteChanRecvSimpleExpr(c *dstutil.Cursor) {
	// read from chan in simple form
	// <-ch
	if recvExpr, ok := isRecvExpr(c.Node()); ok {
		// we catch (val, ok) cases earlier
		typ, _ := t.isChanType(recvExpr.X)

		c.Replace(&dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   t.maybeExtractNamedChanType(recvExpr.X, typ),
				Sel: dst.NewIdent("Recv"),
			},
		})
	}
}

func (t *packageTranslator) rewriteChanRecvOk(c *dstutil.Cursor) {
	// read from chan in val, ok = <-ch

	rhs, ok := isDualAssign(c)
	if !ok {
		return
	}

	if recvExpr, ok := isRecvExpr(*rhs); ok {
		*rhs = &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   recvExpr.X,
				Sel: dst.NewIdent("RecvOk"),
			},
		}
	}
}

func (t *packageTranslator) rewriteChanClose(c *dstutil.Cursor) {
	// close ch
	if callExpr, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(callExpr.Fun, "close") {
		typ, _ := t.isChanType(callExpr.Args[0])
		r := &dst.CallExpr{
			Decs: callExpr.Decs,
			Fun: &dst.SelectorExpr{
				X:   t.maybeExtractNamedChanType(callExpr.Args[0], typ),
				Sel: dst.NewIdent("Close"),
			},
		}
		c.Replace(r)
	}
}

func (t *packageTranslator) rewriteChanSend(c *dstutil.Cursor) {
	// write to chan
	// ch <-
	if sendStmt, ok := c.Node().(*dst.SendStmt); ok {
		typ, _ := t.isChanType(sendStmt.Chan)
		c.Replace(&dst.ExprStmt{
			X: &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   t.maybeExtractNamedChanType(sendStmt.Chan, typ),
					Sel: dst.NewIdent("Send"),
				},
				Args: []dst.Expr{
					sendStmt.Value,
				},
			},
		})
	}
}

func (t *packageTranslator) rewriteChanLiteral(c *dstutil.Cursor) {
	if ident, ok := c.Node().(*dst.Ident); ok && ident.Name == "nil" {
		if chanType, ok := t.isChanType(ident); ok {
			c.Replace(&dst.CallExpr{
				Fun: &dst.IndexListExpr{
					X: t.newRuntimeSelector("NilChan"),
					Indices: []dst.Expr{
						t.makeTypeExpr(chanType.Type.Elem()),
					},
				},
			})
		}
	}
}

func (t *packageTranslator) rewriteMakeChan(c *dstutil.Cursor) {
	// make(chan)
	if makeExpr, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(makeExpr.Fun, "make") {
		if typ, ok := t.isChanType(makeExpr); ok {
			var lenArg dst.Expr = &dst.BasicLit{Kind: token.INT, Value: "0"}
			if len(makeExpr.Args) >= 2 {
				lenArg = t.apply(makeExpr.Args[1]).(dst.Expr)
			}

			c.Replace(t.maybeConvertChanType(&dst.CallExpr{
				Fun: &dst.IndexListExpr{
					X: t.newRuntimeSelector("NewChan"),
					Indices: []dst.Expr{
						t.makeTypeExpr(typ.Type.Elem()),
					},
				},
				Args: []dst.Expr{
					lenArg,
				},
			}, typ))
		}
	}
}

func (t *packageTranslator) rewriteSelectStmt(c *dstutil.Cursor) {
	// select stmt
	if selectStmt, ok := c.Node().(*dst.SelectStmt); ok {
		handlers := []dst.Expr{}
		var clauses []dst.Stmt

		copyStmt := &dst.AssignStmt{
			Lhs: []dst.Expr{},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{},
		}

		suffix := t.suffix()
		selectIdx := "idx" + suffix
		selectVal := "val" + suffix
		valUsed := false
		selectOk := "ok" + suffix
		okUsed := false

		hasDefault := false

		counter := 0

		for _, clause := range selectStmt.Body.List {
			clause := clause.(*dst.CommClause)

			isDefault := false
			var handler dst.Expr

			if sendStmt, ok := clause.Comm.(*dst.SendStmt); ok {
				handler = &dst.CallExpr{
					Fun:  &dst.SelectorExpr{X: sendStmt.Chan, Sel: dst.NewIdent("SendSelector")},
					Args: []dst.Expr{sendStmt.Value},
				}
			} else if exprStmt, ok := clause.Comm.(*dst.ExprStmt); ok {
				recvExpr, ok := isRecvExpr(exprStmt.X)
				if !ok {
					log.Fatal("bad recv expr")
				}
				typ, _ := t.isChanType(recvExpr.X)

				handler = &dst.CallExpr{
					Fun: &dst.SelectorExpr{X: t.maybeExtractNamedChanType(recvExpr.X, typ), Sel: dst.NewIdent("RecvSelector")},
				}
			} else if assignStmt, ok := clause.Comm.(*dst.AssignStmt); ok {
				if len(assignStmt.Rhs) != 1 {
					log.Fatal("bad recv assign stmt")
				}
				recvExpr, ok := isRecvExpr(assignStmt.Rhs[0])
				if !ok {
					log.Fatal("bad recv expr")
				}

				typ, _ := t.isChanType(recvExpr.X)

				lhs := assignStmt.Lhs
				rhs := []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.IndexExpr{
							X:     t.newRuntimeSelector("ChanCast"),
							Index: t.makeTypeExpr(typ.Type.Elem()),
						},
						Args: []dst.Expr{dst.NewIdent(selectVal)},
					},
				}
				valUsed = true
				if len(lhs) > 1 {
					rhs = append(rhs, dst.NewIdent(selectOk))
					okUsed = true
				}

				clause.Body = append([]dst.Stmt{
					&dst.AssignStmt{
						Lhs: lhs,
						Tok: assignStmt.Tok,
						Rhs: rhs,
						Decs: dst.AssignStmtDecorations{
							NodeDecs: dst.NodeDecs{
								End: clause.Decs.Comm,
							},
						},
					},
				}, clause.Body...)

				handler = &dst.CallExpr{
					Fun: &dst.SelectorExpr{X: recvExpr.X, Sel: dst.NewIdent("RecvSelector")},
				}
			} else if clause.Comm == nil {
				hasDefault = true
				isDefault = true
			} else {
				panic("help")
			}

			var caseList []dst.Expr
			if !isDefault {
				caseList = []dst.Expr{&dst.BasicLit{Kind: token.INT, Value: fmt.Sprint(counter)}}
				counter++
				handlers = append(handlers, handler)
			}
			clauses = append(clauses, &dst.CaseClause{
				List: caseList,
				Body: clause.Body,
				Decs: dst.CaseClauseDecorations{
					NodeDecs: clause.Decs.NodeDecs,
					Case:     clause.Decs.Case,
					Colon:    clause.Decs.Colon,
				},
			})
		}

		if !hasDefault {
			// output a default case to help the compiler understand we will never pass over this select
			clauses = append(clauses, &dst.CaseClause{
				// stick this case at the original closing bracket to help comments find their way
				List: nil, // default
				Body: []dst.Stmt{
					&dst.ExprStmt{
						X: &dst.CallExpr{
							Fun: dst.NewIdent("panic"),
							Args: []dst.Expr{
								&dst.BasicLit{
									Kind:  token.STRING,
									Value: strconv.Quote("unreachable select"),
								},
							},
						},
					},
				},
			})
		} else {
			handlers = append(handlers, &dst.CallExpr{
				Fun: t.newRuntimeSelector("DefaultSelector"),
			})
		}

		if len(copyStmt.Lhs) > 0 {
			c.InsertBefore(copyStmt)
		}

		if !valUsed {
			selectVal = "_"
		}
		if !okUsed {
			selectOk = "_"
		}

		var call dst.Expr
		if hasDefault && counter == 1 {
			// special case
			call = handlers[0]
			sel := call.(*dst.CallExpr).Fun.(*dst.SelectorExpr).Sel
			sel.Name = "Select" + strings.TrimSuffix(sel.Name, "Selector") + "OrDefault"
		} else {
			call = &dst.CallExpr{
				Fun:  t.newRuntimeSelector("Select"),
				Args: handlers,
			}
		}

		replacement := &dst.SwitchStmt{
			Decs: dst.SwitchStmtDecorations{
				NodeDecs: selectStmt.Decs.NodeDecs,
				Switch:   selectStmt.Decs.Select,
			},
			Init: &dst.AssignStmt{
				Lhs: []dst.Expr{
					dst.NewIdent(selectIdx),
					dst.NewIdent(selectVal),
					dst.NewIdent(selectOk),
				},
				Tok: token.DEFINE,
				Rhs: []dst.Expr{call},
			},
			Tag: dst.NewIdent(selectIdx),
			Body: &dst.BlockStmt{
				List: clauses,
			},
		}
		c.Replace(replacement)
	}
}
