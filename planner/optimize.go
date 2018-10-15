// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package planner

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/tidb/infoschema"
	plannercore "github.com/pingcap/tidb/planner/core"
	"github.com/pingcap/tidb/privilege"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pkg/errors"
)

// Optimize does optimization and creates a Plan.
// The node must be prepared first.
func Optimize(ctx sessionctx.Context, node ast.Node, is infoschema.InfoSchema, plan plannercore.Plan) (plannercore.Plan, error) {
	fp := plannercore.TryFastPlan(ctx, node)
	if fp != nil {
		return fp, nil
	}

	var p plannercore.Plan
	var err error
	builder := plannercore.NewPlanBuilder(ctx, is)
	if plan == nil {
		// build logical plan
		ctx.GetSessionVars().PlanID = 0
		ctx.GetSessionVars().PlanColumnID = 0
		p, err = builder.Build(node)
		if err != nil {
			return nil, err
		}
	} else {
		p = plan
	}

	// Check privilege. Maybe it's better to move this to the Preprocess, but
	// we need the table information to check privilege, which is collected
	// into the visitInfo in the logical plan builder.
	if pm := privilege.GetPrivilegeManager(ctx); pm != nil {
		if !plannercore.CheckPrivilege(pm, builder.GetVisitInfo()) {
			return nil, errors.New("privilege check fail")
		}
	}

	// Handle the execute statement.
	if execPlan, ok := p.(*plannercore.Execute); ok {
		err := execPlan.OptimizePreparedPlan(ctx, is)
		return p, errors.Trace(err)
	}

	// Handle the non-logical plan statement.
	logic, isLogicalPlan := p.(plannercore.LogicalPlan)
	if !isLogicalPlan {
		return p, nil
	}

	// Handle the logical plan statement, use cascades planner if enabled.
	if ctx.GetSessionVars().EnableCascadesPlanner {
		return nil, errors.New("the cascades planner is not implemented yet")
	}
	return plannercore.DoOptimize(builder.GetOptFlag(), logic)
}

func init() {
	plannercore.OptimizeAstNode = Optimize
}
