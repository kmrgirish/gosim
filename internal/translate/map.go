package translate

import (
	"go/ast"
	"go/token"
	"go/types"
	"log"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

type MapType struct {
	Type  *types.Map
	Named types.Type
}

func isTypMapType(typ types.Type) (MapType, bool) {
	if mapTyp, ok := typ.Underlying().(*types.Map); ok {
		return MapType{Type: mapTyp, Named: typ}, true
	}

	if param, ok := typ.(*types.TypeParam); ok {
		iface := param.Constraint().Underlying().(*types.Interface)
		if iface.NumEmbeddeds() == 1 {
			if union, ok := iface.EmbeddedType(0).(*types.Union); ok {
				if union.Len() == 1 {
					if mapType, ok := union.Term(0).Type().(*types.Map); ok {
						return MapType{Type: mapType, Named: typ}, true
					}
				}
			}
		}
	}

	return MapType{}, false
}

func (t *packageTranslator) isMapType(expr dst.Expr) (MapType, bool) {
	if astExpr, ok := t.astMap.Nodes[expr].(ast.Expr); ok {
		if convertedType, ok := t.implicitConversions[astExpr]; ok {
			// check here in case we pass a map[string]string to an any field
			if typ, ok := isTypMapType(convertedType); ok {
				return typ, ok
			}
		}
	}

	if typ, ok := t.getType(expr); ok {
		return isTypMapType(typ)
	}

	return MapType{}, false
}

func (t *packageTranslator) isMapIndex(node dst.Node) (*dst.IndexExpr, MapType, bool) {
	expr, ok := node.(dst.Expr)
	if !ok {
		return nil, MapType{}, false
	}
	expr = dstutil.Unparen(expr)
	if indexExpr, ok := expr.(*dst.IndexExpr); ok {
		if mapType, ok := t.isMapType(indexExpr.X); ok {
			return indexExpr, mapType, true
		}
	}
	return nil, MapType{}, false
}

func (t *packageTranslator) maybeExtractNamedMapType(x dst.Expr, typ MapType) dst.Expr {
	if typ.Type == typ.Named {
		return x
	}
	return &dst.CallExpr{Fun: t.newRuntimeSelector("ExtractMap"), Args: []dst.Expr{x}}
}

func (t *packageTranslator) maybeConvertMapType(x dst.Expr, typ MapType) dst.Expr {
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

func (t *packageTranslator) rewriteMapRange(c *dstutil.Cursor) {
	// map range
	if rangeStmt, ok := c.Node().(*dst.RangeStmt); ok {
		if typ, ok := t.isMapType(rangeStmt.X); ok {
			rangeStmt.X = &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   t.maybeExtractNamedMapType(rangeStmt.X, typ),
					Sel: dst.NewIdent("Range"),
				},
			}
		}
	}
}

func (t *packageTranslator) rewriteMapDelete(c *dstutil.Cursor) {
	// delete from map
	if callExpr, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(callExpr.Fun, "delete") {
		if typ, ok := t.isMapType(callExpr.Args[0]); ok {
			c.Replace(&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   t.maybeExtractNamedMapType(t.apply(callExpr.Args[0]).(dst.Expr), typ),
					Sel: dst.NewIdent("Delete"),
				},
				Args: []dst.Expr{
					t.apply(callExpr.Args[1]).(dst.Expr),
				},
			})
		}
	}
}

func (t *packageTranslator) rewriteMapClear(c *dstutil.Cursor) {
	// delete from map
	if callExpr, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(callExpr.Fun, "clear") {
		if typ, ok := t.isMapType(callExpr.Args[0]); ok {
			c.Replace(&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   t.maybeExtractNamedMapType(t.apply(callExpr.Args[0]).(dst.Expr), typ),
					Sel: dst.NewIdent("Clear"),
				},
			})
		}
	}
}

func (t *packageTranslator) rewriteMakeMap(c *dstutil.Cursor) {
	if makeExpr, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(makeExpr.Fun, "make") {
		if typ, ok := t.isMapType(makeExpr); ok {
			c.Replace(t.maybeConvertMapType(&dst.CallExpr{
				Fun: &dst.IndexListExpr{
					X: t.newRuntimeSelector("NewMap"),
					Indices: []dst.Expr{
						t.makeTypeExpr(typ.Type.Key()),
						t.makeTypeExpr(typ.Type.Elem()),
					},
				},
				Args: []dst.Expr{},
			}, typ))
		}
	}
}

func (t *packageTranslator) rewriteMapLiteral(c *dstutil.Cursor) {
	// map literal
	if ident, ok := c.Node().(*dst.Ident); ok && ident.Name == "nil" {
		if mapType, ok := t.isMapType(ident); ok {
			c.Replace(t.maybeConvertMapType(&dst.CallExpr{
				Fun: &dst.IndexListExpr{
					X: t.newRuntimeSelector("NilMap"),
					Indices: []dst.Expr{
						t.makeTypeExpr(mapType.Type.Key()),
						t.makeTypeExpr(mapType.Type.Elem()),
					},
				},
			}, mapType))
		}
	} else if compositeLit, ok := c.Node().(*dst.CompositeLit); ok {
		if typ, ok := t.isMapType(compositeLit); ok {
			var pairs []dst.Expr
			for _, elt := range compositeLit.Elts {
				kv, ok := elt.(*dst.KeyValueExpr)
				if !ok {
					log.Fatal("map composite lit elt is not keyvalueexpr")
				}
				nodeDecs := kv.Decs.NodeDecs
				kv.Decs.NodeDecs = dst.NodeDecs{
					Start: nodeDecs.Start,
				}
				nodeDecs.Start = nil

				key := t.apply(kv.Key).(dst.Expr)
				value := t.apply(kv.Value).(dst.Expr)

				if nestedLit, ok := key.(*dst.CompositeLit); ok && nestedLit.Type == nil {
					nestedLit.Type = t.makeTypeExpr(typ.Type.Key())
				}
				if nestedLit, ok := value.(*dst.CompositeLit); ok && nestedLit.Type == nil {
					// XXX: janky hack
					if ptr, ok := typ.Type.Elem().Underlying().(*types.Pointer); ok {
						nestedLit.Type = t.makeTypeExpr(ptr.Elem())
						value = &dst.UnaryExpr{
							Op: token.AND,
							X:  value,
						}
					} else {
						nestedLit.Type = t.makeTypeExpr(typ.Type.Elem())
					}
				}

				pairs = append(pairs, &dst.CompositeLit{
					Elts: []dst.Expr{
						&dst.KeyValueExpr{
							Key:   dst.NewIdent("K"),
							Value: key,
						},
						&dst.KeyValueExpr{
							Key:   dst.NewIdent("V"),
							Value: value,
						},
					},
					Decs: dst.CompositeLitDecorations{
						NodeDecs: nodeDecs,
					},
				})
			}

			c.Replace(t.maybeConvertMapType(&dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X: &dst.CompositeLit{
						Type: &dst.IndexListExpr{
							X: t.newRuntimeSelector("MapLiteral"),
							Indices: []dst.Expr{
								t.makeTypeExpr(typ.Type.Key()),
								t.makeTypeExpr(typ.Type.Elem()),
							},
						},
						Elts: pairs,
					},
					Sel: dst.NewIdent("Build"),
				},
			}, typ))
		}
	} else if expr, ok := c.Node().(dst.Expr); ok {
		astExpr, _ := t.astMap.Nodes[expr].(ast.Expr)
		if typ, ok := t.implicitConversions[astExpr]; ok {
			if _, ok := isTypMapType(typ); ok {
				c.Replace(&dst.CallExpr{
					Fun:  t.makeTypeExpr(typ),
					Args: []dst.Expr{expr},
				})
			}
		}
	}
}

func (t *packageTranslator) rewriteMapLen(c *dstutil.Cursor) {
	if lenExpr, ok := c.Node().(*dst.CallExpr); ok && t.isNamedBuiltIn(lenExpr.Fun, "len") {
		if typ, ok := t.isMapType(lenExpr.Args[0]); ok {
			// XXX: len, cap chan?
			lenExpr.Fun = &dst.SelectorExpr{
				X:   t.maybeExtractNamedMapType(lenExpr.Args[0], typ),
				Sel: dst.NewIdent("Len"),
			}
			lenExpr.Args = nil
		}
	}
}

func (t *packageTranslator) rewriteMapType(c *dstutil.Cursor) {
	// map type
	if mapType, ok := c.Node().(*dst.MapType); ok {
		c.Replace(&dst.IndexListExpr{
			X: t.newRuntimeSelector("Map"),
			Indices: []dst.Expr{
				mapType.Key,
				t.apply(mapType.Value).(dst.Expr),
			},
		})
	}

	// for generics, eject the outer ~ on map...
	if unary, ok := c.Node().(*dst.UnaryExpr); ok && unary.Op == token.TILDE {
		if mapType, ok := unary.X.(*dst.MapType); ok {
			c.Replace(&dst.IndexListExpr{
				X: t.newRuntimeSelector("Map"),
				Indices: []dst.Expr{
					mapType.Key,
					t.apply(mapType.Value).(dst.Expr),
				},
			})
		}
	}
}

func (t *packageTranslator) rewriteMapIndex(c *dstutil.Cursor) {
	if indexExpr, wrapped, ok := t.isMapIndex(c.Node()); ok {
		// we catch (val, ok) cases earlier
		c.Replace(&dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   t.maybeExtractNamedMapType(t.apply(indexExpr.X).(dst.Expr), wrapped),
				Sel: dst.NewIdent("Get"),
			},
			Args: []dst.Expr{
				t.apply(indexExpr.Index).(dst.Expr),
			},
		})
	}
}

func (t *packageTranslator) rewriteMapGetOk(c *dstutil.Cursor) {
	// get from map in v, ok = m[k]

	rhs, ok := isDualAssign(c)
	if !ok {
		return
	}

	if index, wrapped, ok := t.isMapIndex(*rhs); ok {
		*rhs = &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X:   t.maybeExtractNamedMapType(index.X, wrapped),
				Sel: dst.NewIdent("GetOk"),
			},
			Args: []dst.Expr{
				index.Index,
			},
		}
	}
}

func isOpAssign(tok token.Token) (token.Token, bool) {
	ops := map[token.Token]token.Token{
		token.ADD_ASSIGN:     token.ADD,
		token.SUB_ASSIGN:     token.SUB,
		token.MUL_ASSIGN:     token.MUL,
		token.QUO_ASSIGN:     token.QUO,
		token.REM_ASSIGN:     token.REM,
		token.AND_ASSIGN:     token.AND,
		token.OR_ASSIGN:      token.OR,
		token.XOR_ASSIGN:     token.XOR,
		token.SHL_ASSIGN:     token.SHL,
		token.SHR_ASSIGN:     token.SHR,
		token.AND_NOT_ASSIGN: token.AND_NOT,
	}
	out, ok := ops[tok]
	return out, ok
}

type mapModifyInfo struct {
	copyVarsStmt dst.Stmt
	writeExpr    dst.Expr
}

type mapAssignInfo struct {
	mapModify func(op token.Token, val dst.Expr) mapModifyInfo
	mapSet    func(val dst.Expr) dst.Expr
}

func (t *packageTranslator) isMapAssign(lhs dst.Expr) (mapAssignInfo, bool) {
	if index, wrapped, ok := t.isMapIndex(lhs); ok {
		return mapAssignInfo{
			mapModify: func(op token.Token, val dst.Expr) mapModifyInfo {
				suffix := t.suffix()
				keyName := "key" + suffix
				mapName := "map" + suffix
				return mapModifyInfo{
					copyVarsStmt: &dst.AssignStmt{
						Lhs: []dst.Expr{dst.NewIdent(mapName), dst.NewIdent(keyName)},
						Tok: token.DEFINE,
						Rhs: []dst.Expr{t.apply(index.X).(dst.Expr), t.apply(index.Index).(dst.Expr)},
					},
					writeExpr: &dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   dst.NewIdent(mapName),
							Sel: dst.NewIdent("Set"),
						},
						Args: []dst.Expr{
							dst.NewIdent(keyName),
							&dst.BinaryExpr{
								X: &dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   dst.NewIdent(mapName),
										Sel: dst.NewIdent("Get"),
									},
									Args: []dst.Expr{
										dst.NewIdent(keyName),
									},
								},
								Op: op,
								Y:  t.apply(val).(dst.Expr),
							},
						},
					},
				}
			},
			mapSet: func(val dst.Expr) dst.Expr {
				return &dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   t.maybeExtractNamedMapType(t.apply(index.X).(dst.Expr), wrapped),
						Sel: dst.NewIdent("Set"),
					},
					Args: []dst.Expr{
						t.apply(index.Index).(dst.Expr),
						t.apply(val).(dst.Expr),
					},
				}
			},
		}, true
	}
	return mapAssignInfo{}, false
}

func (t *packageTranslator) rewriteMapAssign(c *dstutil.Cursor) {
	// map assignment
	// "m[x] ="
	if assignStmt, ok := c.Node().(*dst.AssignStmt); ok {
		hasAnySpecial := false
		for _, lhs := range assignStmt.Lhs {
			if _, ok := t.isMapAssign(lhs); ok {
				hasAnySpecial = true
			}
		}
		if hasAnySpecial && len(assignStmt.Lhs) > 1 {
			// XXX: could fix this by introducing temp values and then doing sets right after...
			log.Fatal("do not support map or global assign to multiple values...")
		}
		if !hasAnySpecial {
			return
		}

		info, _ := t.isMapAssign(assignStmt.Lhs[0])

		if op, ok := isOpAssign(assignStmt.Tok); ok {
			info2 := info.mapModify(op, assignStmt.Rhs[0])
			c.InsertBefore(info2.copyVarsStmt)
			c.Replace(&dst.ExprStmt{
				Decs: dst.ExprStmtDecorations{
					NodeDecs: assignStmt.Decs.NodeDecs,
				},
				X: info2.writeExpr,
			})
		} else {
			c.Replace(&dst.ExprStmt{
				Decs: dst.ExprStmtDecorations{
					NodeDecs: assignStmt.Decs.NodeDecs,
				},
				X: info.mapSet(assignStmt.Rhs[0]),
			})
		}
	}

	if incDecStmt, ok := c.Node().(*dst.IncDecStmt); ok {
		info, ok := t.isMapAssign(incDecStmt.X)
		if !ok {
			return
		}

		op := map[token.Token]token.Token{
			token.INC: token.ADD,
			token.DEC: token.SUB,
		}[incDecStmt.Tok]

		info2 := info.mapModify(op, &dst.BasicLit{Kind: token.INT, Value: "1"})
		c.InsertBefore(info2.copyVarsStmt)
		c.Replace(&dst.ExprStmt{
			Decs: dst.ExprStmtDecorations{
				NodeDecs: incDecStmt.Decs.NodeDecs,
			},
			X: info2.writeExpr,
		})
	}
}
