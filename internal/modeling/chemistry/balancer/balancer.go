// Package balancer 化学方程式配平：使用有理数高斯消元 + 整数化。
//
// 输入：人类可读方程 `2NaOH + H2SO4 -> Na2SO4 + 2H2O` 或省略系数 `NaOH + H2SO4 -> Na2SO4 + H2O`。
// 输出：每个物种的最小整数系数。
//
// 算法：
//
//  1. 拆解每个物种为元素 → 计数 map（支持嵌套括号、电荷标记可选）；
//  2. 建立 (#元素 × #物种) 的稀疏矩阵 A，反应物列符号为正、产物列为负；
//  3. 高斯消元（math/big.Rat）求齐次系 Ax=0 的最小整数正解；
//  4. 若发现矩阵秩 = 物种数（无非零零空间）则返回 ErrInvalidEquation。
package balancer

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	// ErrInvalidEquation 等式两侧元素守恒不可解。
	ErrInvalidEquation = errors.New("equation cannot be balanced: not mass-conserving")
	// ErrParse 字面解析失败。
	ErrParse = errors.New("failed to parse equation")
)

// Equation 解析后的反应方程，未配平时所有 Coef 默认 1。
type Equation struct {
	Reactants []SpeciesCount
	Products  []SpeciesCount
	Arrow     string // 原始箭头：→ / -> / = / ⇌
}

// SpeciesCount 单个物种的元素计数。
type SpeciesCount struct {
	Raw      string         // 原始字符串："NaOH"
	Formula  string         // 去掉前置系数后的式子
	Coef     int            // 配平后的整数系数
	Elements map[string]int // 元素 → 原子数
}

var (
	// 中间或末尾允许的箭头变体
	arrowRE = regexp.MustCompile(`→|->|⇌|=`)
	tokenRE = regexp.MustCompile(`([A-Z][a-z]?)(\d*)|\(|\)\d*`)
)

// Parse 解析方程字符串，返回 Equation。系数若以前置整数出现会被合并到 Coef。
func Parse(s string) (*Equation, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("%w: empty", ErrParse)
	}

	arrowLoc := arrowRE.FindStringIndex(s)
	if arrowLoc == nil {
		return nil, fmt.Errorf("%w: missing arrow", ErrParse)
	}
	arrow := s[arrowLoc[0]:arrowLoc[1]]
	left := strings.TrimSpace(s[:arrowLoc[0]])
	right := strings.TrimSpace(s[arrowLoc[1]:])
	if left == "" || right == "" {
		return nil, fmt.Errorf("%w: missing side", ErrParse)
	}

	lhs, err := parseSide(left)
	if err != nil {
		return nil, err
	}
	rhs, err := parseSide(right)
	if err != nil {
		return nil, err
	}
	return &Equation{Reactants: lhs, Products: rhs, Arrow: arrow}, nil
}

func parseSide(s string) ([]SpeciesCount, error) {
	parts := strings.Split(s, "+")
	out := make([]SpeciesCount, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		coef, formula := splitLeadingCoef(p)
		elems, err := parseFormula(formula)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrParse, formula, err)
		}
		out = append(out, SpeciesCount{Raw: p, Formula: formula, Coef: coef, Elements: elems})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: empty side", ErrParse)
	}
	return out, nil
}

func splitLeadingCoef(p string) (int, string) {
	i := 0
	for i < len(p) && p[i] >= '0' && p[i] <= '9' {
		i++
	}
	if i == 0 {
		return 1, p
	}
	n, err := strconv.Atoi(p[:i])
	if err != nil || n <= 0 {
		return 1, p
	}
	return n, strings.TrimSpace(p[i:])
}

// parseFormula 把化学式拆成元素 → 原子数 map。支持单层小括号 `Ca(OH)2`。
// 不处理电荷符号（[H+] 等），但 +/- 后缀会被忽略。
func parseFormula(f string) (map[string]int, error) {
	out := map[string]int{}
	if f == "" {
		return nil, errors.New("empty formula")
	}
	// 去掉电荷标记：H+, OH-, Cu2+, SO4^2-, [H+]
	f = stripCharge(f)

	i := 0
	for i < len(f) {
		ch := f[i]
		switch {
		case ch == '(':
			// 找配对的右括号
			depth := 1
			j := i + 1
			for j < len(f) && depth > 0 {
				if f[j] == '(' {
					depth++
				} else if f[j] == ')' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			if depth != 0 {
				return nil, errors.New("unmatched parenthesis")
			}
			inner, err := parseFormula(f[i+1 : j])
			if err != nil {
				return nil, err
			}
			// 读取右括号后面的数字
			k := j + 1
			for k < len(f) && f[k] >= '0' && f[k] <= '9' {
				k++
			}
			mult := 1
			if k > j+1 {
				m, err := strconv.Atoi(f[j+1 : k])
				if err == nil && m > 0 {
					mult = m
				}
			}
			for e, n := range inner {
				out[e] += n * mult
			}
			i = k
		case unicode.IsUpper(rune(ch)):
			// 元素：大写 + 可选小写
			j := i + 1
			if j < len(f) && unicode.IsLower(rune(f[j])) {
				j++
			}
			elem := f[i:j]
			// 元素后面的数字
			k := j
			for k < len(f) && f[k] >= '0' && f[k] <= '9' {
				k++
			}
			n := 1
			if k > j {
				m, err := strconv.Atoi(f[j:k])
				if err == nil && m > 0 {
					n = m
				}
			}
			out[elem] += n
			i = k
		case ch == ' ' || ch == '\t' || ch == '·':
			i++
		default:
			return nil, fmt.Errorf("unexpected char %q in %q", ch, f)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no element parsed")
	}
	return out, nil
}

func stripCharge(f string) string {
	// 去掉成对方括号
	f = strings.TrimPrefix(f, "[")
	f = strings.TrimSuffix(f, "]")
	// 去掉末尾的 ^xx 或 + / -
	if idx := strings.IndexAny(f, "^"); idx >= 0 {
		f = f[:idx]
	}
	for strings.HasSuffix(f, "+") || strings.HasSuffix(f, "-") {
		f = f[:len(f)-1]
		// 还可能附带数字：Cu2+
		for len(f) > 0 && f[len(f)-1] >= '0' && f[len(f)-1] <= '9' {
			f = f[:len(f)-1]
		}
	}
	return f
}

// Balance 配平给定方程，写入每个 Species.Coef 的最小正整数解。
//
// 算法：构造 (#元素) 行 × (#物种) 列的有理矩阵 A，反应物列正、产物列负；
// 求 Ax=0 的非零正整数解 x（系数）。使用增广高斯消元后回代求自由变量。
func Balance(eq *Equation) error {
	if eq == nil {
		return errors.New("nil equation")
	}
	all := append([]SpeciesCount{}, eq.Reactants...)
	all = append(all, eq.Products...)
	if len(all) < 2 {
		return ErrInvalidEquation
	}

	// 元素列表（按首次出现顺序）
	seen := map[string]bool{}
	elements := []string{}
	for _, sp := range all {
		for e := range sp.Elements {
			if !seen[e] {
				seen[e] = true
				elements = append(elements, e)
			}
		}
	}
	rows := len(elements)
	cols := len(all)
	mat := make([][]*big.Rat, rows)
	for i := range mat {
		mat[i] = make([]*big.Rat, cols)
		for j := 0; j < cols; j++ {
			mat[i][j] = big.NewRat(0, 1)
		}
	}
	for ci, sp := range all {
		sign := int64(1)
		if ci >= len(eq.Reactants) {
			sign = -1
		}
		for ri, e := range elements {
			if n, ok := sp.Elements[e]; ok {
				mat[ri][ci] = big.NewRat(sign*int64(n), 1)
			}
		}
	}

	coefs, err := nullspaceRationalPositive(mat, rows, cols)
	if err != nil {
		return err
	}
	if len(coefs) != cols {
		return ErrInvalidEquation
	}
	for i := range eq.Reactants {
		eq.Reactants[i].Coef = coefs[i]
	}
	for i := range eq.Products {
		eq.Products[i].Coef = coefs[len(eq.Reactants)+i]
	}
	return nil
}

// nullspaceRationalPositive 高斯消元后返回 1 维零空间的最小正整数基。
func nullspaceRationalPositive(mat [][]*big.Rat, rows, cols int) ([]int, error) {
	// 复制
	a := make([][]*big.Rat, rows)
	for i := range mat {
		a[i] = make([]*big.Rat, cols)
		for j := 0; j < cols; j++ {
			a[i][j] = new(big.Rat).Set(mat[i][j])
		}
	}
	// 高斯消元（行简化阶梯型 RREF）
	pivotCols := []int{}
	row := 0
	for col := 0; col < cols && row < rows; col++ {
		// 找主元
		piv := -1
		for r := row; r < rows; r++ {
			if a[r][col].Sign() != 0 {
				piv = r
				break
			}
		}
		if piv == -1 {
			continue
		}
		a[row], a[piv] = a[piv], a[row]
		// 归一化主行
		inv := new(big.Rat).Inv(a[row][col])
		for j := col; j < cols; j++ {
			a[row][j] = new(big.Rat).Mul(a[row][j], inv)
		}
		// 消去其他行
		for r := 0; r < rows; r++ {
			if r == row || a[r][col].Sign() == 0 {
				continue
			}
			factor := new(big.Rat).Set(a[r][col])
			for j := col; j < cols; j++ {
				prod := new(big.Rat).Mul(factor, a[row][j])
				a[r][j] = new(big.Rat).Sub(a[r][j], prod)
			}
		}
		pivotCols = append(pivotCols, col)
		row++
	}
	freeCols := []int{}
	pivotSet := map[int]bool{}
	for _, c := range pivotCols {
		pivotSet[c] = true
	}
	for c := 0; c < cols; c++ {
		if !pivotSet[c] {
			freeCols = append(freeCols, c)
		}
	}
	if len(freeCols) == 0 {
		return nil, ErrInvalidEquation
	}
	// 取一维零空间：把第一个自由变量设为 1，其它自由变量设为 0。
	x := make([]*big.Rat, cols)
	for i := range x {
		x[i] = big.NewRat(0, 1)
	}
	x[freeCols[0]] = big.NewRat(1, 1)
	// 把其他自由变量保持 0 — 多于 1 个自由变量时只用第一组解。
	// 回代：对每个 pivot 行，x[pivotCol] = -sum(coef[freeCol]*x[freeCol]) for freeCols
	for i, pc := range pivotCols {
		sum := big.NewRat(0, 1)
		for c := pc + 1; c < cols; c++ {
			if pivotSet[c] {
				continue
			}
			term := new(big.Rat).Mul(a[i][c], x[c])
			sum.Add(sum, term)
		}
		x[pc] = new(big.Rat).Neg(sum)
	}
	// 全部 → 正且整数化
	// 找最小公倍数将分母清掉
	lcm := big.NewInt(1)
	for _, r := range x {
		d := r.Denom()
		if d.Sign() == 0 {
			continue
		}
		lcm = lcmBig(lcm, d)
	}
	ints := make([]int, cols)
	for i, r := range x {
		n := new(big.Int).Mul(r.Num(), lcm)
		n.Quo(n, r.Denom())
		v := n.Int64()
		if v == 0 {
			// 0 表示该物种系数为 0 → 方程不可配平
			return nil, ErrInvalidEquation
		}
		if v < 0 {
			v = -v
		}
		ints[i] = int(v)
	}
	// 全部都是正数：用 GCD 缩小到最简
	g := ints[0]
	for _, v := range ints[1:] {
		g = gcd(g, v)
	}
	if g > 1 {
		for i := range ints {
			ints[i] /= g
		}
	}
	return ints, nil
}

func gcd(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func lcmBig(a, b *big.Int) *big.Int {
	if a.Sign() == 0 || b.Sign() == 0 {
		return big.NewInt(1)
	}
	g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(a), new(big.Int).Abs(b))
	res := new(big.Int).Quo(new(big.Int).Mul(a, b), g)
	return res.Abs(res)
}
