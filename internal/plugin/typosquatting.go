package plugin

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// popularPackages 是仿冒检测的比对基准：PyPI 与 npm 上最常见的包名。
// 仿冒包托管在同一个注册表上，装错一个就等于执行攻击者的代码。
var popularPackages = []string{
	// PyPI
	"requests", "urllib3", "certifi", "idna", "charset-normalizer", "python-dateutil",
	"six", "pytz", "setuptools", "wheel", "numpy", "pandas", "scipy", "matplotlib",
	"seaborn", "scikit-learn", "tensorflow", "torch", "transformers", "django",
	"flask", "fastapi", "starlette", "uvicorn", "pydantic", "sqlalchemy", "celery",
	"redis", "pymongo", "psycopg2", "boto3", "botocore", "pillow", "opencv-python",
	"beautifulsoup4", "lxml", "pyyaml", "cryptography", "paramiko", "httpx",
	"aiohttp", "tqdm", "pytest", "cffi", "grpcio", "protobuf", "pyjwt",
	"python-dotenv", "jinja2", "click", "rich", "typer", "openai",
	// npm
	"express", "lodash", "react", "react-dom", "vue", "next", "svelte", "axios",
	"node-fetch", "got", "chalk", "commander", "yargs", "dotenv", "cors", "helmet",
	"morgan", "body-parser", "cookie-parser", "multer", "socket.io", "ws",
	"jsonwebtoken", "bcrypt", "passport", "mongoose", "sequelize", "prisma",
	"typescript", "eslint", "prettier", "jest", "mocha", "vitest", "webpack",
	"vite", "rollup", "esbuild", "core-js", "tailwindcss", "zod", "uuid",
	"semver", "minimist", "glob", "rimraf", "cross-env", "moment", "dayjs",
	"rxjs", "redux", "zustand",
}

// normalizedPopular 是归一化后的基准表，包级初始化一次。
var normalizedPopular [][2]string // [归一化名, 原名]

func init() {
	for _, name := range popularPackages {
		normalizedPopular = append(normalizedPopular, [2]string{normalizePkgName(name), name})
	}
}

// normalizePkgName 归一化包名：小写、去掉 npm scope 前缀、删除 -_. 分隔符
// （PyPI 对 -_. 视为等价，仿冒者也常用分隔符差异做伪装）。
func normalizePkgName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.ToLower(name)
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == '.' {
			return -1
		}
		return r
	}, name)
}

// editDistanceOSA 是受限 Damerau-Levenshtein 距离（最优对齐）：相比普通
// Levenshtein 把相邻换位算作一次编辑，"reqeusts"→"requests" 才能按 d=1 命中。
func editDistanceOSA(a, b string) int {
	ar, br := []rune(a), []rune(b)
	la, lb := len(ar), len(br)
	prev2 := make([]int, lb+1)
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			m := min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ar[i-1] == br[j-2] && ar[i-2] == br[j-1] {
				if t := prev2[j-2] + 1; t < m {
					m = t
				}
			}
			cur[j] = m
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[lb]
}

type pkgRef struct {
	name string
	line int
}

var (
	reqNameRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)`)
	depBlockRe = regexp.MustCompile(`"(?:dependencies|devDependencies|optionalDependencies|peerDependencies)"\s*:\s*\{([^{}]*)\}`)
	depKeyRe   = regexp.MustCompile(`"([^"]+)"\s*:`)
	poetryKeyRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*=`)
	quotedRe    = regexp.MustCompile(`"([^"]+)"`)
)

func parseRequirements(content string) []pkgRef {
	var out []pkgRef
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") ||
			strings.Contains(trimmed, "://") {
			continue
		}
		if m := reqNameRe.FindStringSubmatch(trimmed); m != nil {
			out = append(out, pkgRef{m[1], i + 1})
		}
	}
	return out
}

func parsePackageJSON(content string) []pkgRef {
	var out []pkgRef
	for _, block := range depBlockRe.FindAllStringSubmatchIndex(content, -1) {
		for _, key := range depKeyRe.FindAllStringSubmatchIndex(content[block[2]:block[3]], -1) {
			out = append(out, pkgRef{
				name: content[block[2]+key[2] : block[2]+key[3]],
				line: lineOf(content, block[2]+key[0]),
			})
		}
	}
	return out
}

func parsePyproject(content string) []pkgRef {
	var out []pkgRef
	section := ""
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			section = trimmed
			continue
		}
		switch {
		case strings.Contains(section, "poetry") && strings.Contains(section, "dependencies"):
			if m := poetryKeyRe.FindStringSubmatch(trimmed); m != nil {
				out = append(out, pkgRef{m[1], i + 1})
			}
		case strings.HasPrefix(section, "[project"):
			// PEP 621：dependencies = ["requests>=2.0", ...]，逐个引号串提取包名。
			for _, q := range quotedRe.FindAllStringSubmatch(trimmed, -1) {
				if m := reqNameRe.FindStringSubmatch(q[1]); m != nil {
					out = append(out, pkgRef{m[1], i + 1})
				}
			}
		}
	}
	return out
}

type TyposquattingCheckPlugin struct{}

func (TyposquattingCheckPlugin) Meta() Meta {
	return Meta{
		ID:          "typosquatting",
		Name:        "Typosquatting Package Names",
		Description: "Detects declared dependencies whose names are one or two edits away from a well-known PyPI or npm package, the signature of typosquatting.",
	}
}

func (TyposquattingCheckPlugin) Scan(skillDir, filePath, content string) []Issue {
	// 只在依赖清单里比对：包名比对对其它文件没有意义。
	base := strings.ToLower(filepath.Base(filePath))
	var candidates []pkgRef
	switch {
	case strings.HasPrefix(base, "requirements") &&
		(strings.HasSuffix(base, ".txt") || strings.HasSuffix(base, ".in")):
		candidates = parseRequirements(content)
	case base == "package.json":
		candidates = parsePackageJSON(content)
	case base == "pyproject.toml":
		candidates = parsePyproject(content)
	default:
		return nil
	}

	var issues []Issue
	rel := relativePath(filePath, skillDir)
	seen := map[string]bool{}
	for _, c := range candidates {
		norm := normalizePkgName(c.name)
		// 过短的名字与基准表比对噪音太大；同一个名字只报一次。
		if len(norm) < 3 || seen[norm] {
			continue
		}
		for _, pop := range normalizedPopular {
			d := editDistanceOSA(norm, pop[0])
			// 相等是正常声明；d=1 是典型仿冒；d=2 只对足够长的名字启用。
			if d == 0 || d > 2 || (d == 2 && len(norm) < 8) {
				continue
			}
			seen[norm] = true
			issues = append(issues, Issue{
				Name:     "Typosquatting package name",
				Severity: SeverityHigh,
				Category: CatNetworkAbuse,
				Description: fmt.Sprintf("Package %q looks similar to popular package %q (edit distance %d); "+
					"installing it would run attacker-controlled code.", c.name, pop[1], d),
				FilePath: rel,
				Line:     c.line,
				// 相似不等于仿冒，是否误伤合法的相近包名由复核判断。
				AuditRequired: true,
			})
			break
		}
	}
	return issues
}
