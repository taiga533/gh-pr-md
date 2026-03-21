
# #12655 feat(pr diff): add --exclude flag to filter files from diff output

**Author:** @yuvrajangadsingh

Adds a --exclude (-e) flag to gh pr diff that lets you filter out files matching glob patterns from the diff output.

**Example usage:**

# Exclude all YAML files
gh pr diff --exclude '*.yml'

# Exclude multiple patterns
gh pr diff --exclude '*.yml' --exclude 'vendor/*'

# Works with --name-only too
gh pr diff --name-only --exclude '*.generated.*'

Patterns are matched against both the full file path and the basename, so --exclude '*.yml' will match dir/file.yml.

Uses Go's filepath.Match for glob support — no new dependencies.

Closes #8739

--------

### @yuvrajangadsingh commented on 2026-02-20 09:41

@BagToad @babakks hey, just wanted to check if this is something the team would be interested in — adds an --exclude
flag
to gh pr diff so you can filter out noisy files (like lockfiles, snapshots, generated code). pretty common ask based on
the issues i've seen. let me know if there's anything to adjust!

### @yuvrajangadsingh commented on 2026-02-25 10:02

gentle bump on this one — would love to get some eyes on it when someone has a chance. happy to make changes if anything
needs adjusting.

### @BagToad requested changes on 2026-03-04 00:14

Nice implementation — the exclude logic is clean and well-tested.

One change needed: the diff header regex is duplicated. changedFilesNames (line ~310) already has an inline regex that's
nearly identical to your new diffHeaderRegexp:

// existing (changedFilesNames):
pattern := regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["']?)b/(.*)`)

// new (extractFileName):
var diffHeaderRegexp = regexp.MustCompile(`diff\s--git.*\s(["']?)b/(.*)`)

Could you share the compiled regex between both and ideally refactor changedFilesNames to use your
splitDiffSections/extractFileName helpers? That would reduce duplication and make the code easier to maintain.

If this is too painful, let me know — it's kind of a nit so I'm happy to tackle it in a follow-up if needed instead.

### @BagToad commented on 2026-03-04 02:19

Also note, we need to fix the linter checks.

### @yuvrajangadsingh commented on 2026-03-04 21:49

@BagToad fixed both — shared the compiled diffHeaderRegexp between changedFilesNames and extractFileName (removed the
inline duplicate), and fixed the gofmt spacing. rebased on trunk too.

### @babakks reviewed on 2026-03-05 16:35

Thanks for the PR, @yuvrajangadsingh! 🙏

Just commented some thoughts, but the main one is the last (about using path instead of filepath).

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -92,6 +95,7 @@ func NewCmdDiff(f *cmdutil.Factory, runF func(*DiffOptions) error) *cobra.Comman
 	cmd.Flags().BoolVar(&opts.Patch, "patch", false, "Display diff in patch format")
 	cmd.Flags().BoolVar(&opts.NameOnly, "name-only", false, "Display only names of changed files")
 	cmd.Flags().BoolVarP(&opts.BrowserMode, "web", "w", false, "Open the pull request diff in the browser")
+	cmd.Flags().StringSliceVarP(&opts.Exclude, "exclude", "e", nil, "Exclude files matching glob `patterns` from the
diff")

We need to add examples to show how to use this option. This is what I recommend:

# See diff for current branch
$ gh pr diff

# See diff for a specific PR
$ gh pr diff 123

# Exclude files from diff output
gh pr diff --exclude '*.yml' --exclude 'generated/*'

# Exclude matching files by name
$ gh pr diff --name-only --exclude '*.generated.*'

See here https://github.com/cli/cli/blob/19d70d1c6bd2d0b25c8ba8a1730a537a9a520a0c/pkg/cmd/pr/list/list.go#L71-L86 on the
formatting of examples.

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)

This can be simplified:

**Suggested change:**

var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s("?)b/(.*)`)

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -292,8 +303,7 @@ func changedFilesNames(w io.Writer, r io.Reader) error {
 	// `"`` + hello-\360\237\230\200-world"
 	//
 	// Where I'm using the `` to indicate a string to avoid confusion with the " character.
-	pattern := regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
-	matches := pattern.FindAllStringSubmatch(string(diff), -1)
+	matches := diffHeaderRegexp.FindAllStringSubmatch(string(diff), -1)

**thought:** we should really think about using a reliable third-party for this stuff. 🤷

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {

**nitpick:** patterns should be named excludePatterns or simply exclude to avoid confusion.

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, patterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	var sections []string
+	for {
+		idx := strings.Index(diff, marker)
+		if idx == -1 {
+			if len(diff) > 0 {
+				sections = append(sections, diff)
+			}
+			break
+		}
+		sections = append(sections, diff[:idx+1]) // include the trailing \n
+		diff = diff[idx+1:]                       // next section starts at "diff --git"
+	}
+	return sections
+}

**nitpick:** wondering if we can replace this with a strings.Split call or even a regexp split: 🤔

func splitDiffSections(diff string) []string {
	marker := "diff --git "
	var sections []string
	for s := range strings.SplitSeq(diff, "\n"+marker) {
		sections = append(sections, marker+s+"\n")
	}
	if len(sections)==1 {
		return []string{diff}
	}
	return sections
}

or

const diffMarker = "diff --git "
var diffMarkerRE = regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(diffMarker))
func splitDiffSections(diff string) []string {
	var sections []string
	for _, s := range diffMarkerRE.Split(diff, -1) {
		sections = append(sections, marker+s)
	}
	return sections
}

I haven't really tried them, but would love to hear if you have any thoughts, @yuvrajangadsingh @BagToad.

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, patterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	var sections []string
+	for {
+		idx := strings.Index(diff, marker)
+		if idx == -1 {
+			if len(diff) > 0 {
+				sections = append(sections, diff)
+			}
+			break
+		}
+		sections = append(sections, diff[:idx+1]) // include the trailing \n
+		diff = diff[idx+1:]                       // next section starts at "diff --git"
+	}
+	return sections
+}
+
+func extractFileName(section []byte) string {
+	m := diffHeaderRegexp.FindSubmatch(section)
+	if m == nil {
+		return ""
+	}
+	return strings.TrimSpace(string(m[1]) + string(m[2]))
+}
+
+func matchesAny(name string, patterns []string) bool {
+	for _, p := range patterns {
+		if matched, _ := filepath.Match(p, name); matched {
+			return true
+		}
+		// Also match against the basename so "*.yml" matches "dir/file.yml"
+		if matched, _ := filepath.Match(p, filepath.Base(name)); matched {
+			return true
+		}
+	}
+	return false
+}

The filepath package behaves differently based on the host OS. Here, for example, filepath.Match will regard \ as a path
separator, so the user cannot escape chars with it. So, the experience would be different in edge cases based on the OS
the user is running gh on.

My suggestion, is to use path.Match to eliminate the OS factor. In that case, we should also indicate that the patterns
should use forward-slash as separator.

WDYT?

### @BagToad reviewed on 2026-03-05 16:53

#### @BagToad commented on pkg/cmd/pr/diff/diff.go

@@ -292,8 +303,7 @@ func changedFilesNames(w io.Writer, r io.Reader) error {
 	// `"`` + hello-\360\237\230\200-world"
 	//
 	// Where I'm using the `` to indicate a string to avoid confusion with the " character.
-	pattern := regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
-	matches := pattern.FindAllStringSubmatch(string(diff), -1)
+	matches := diffHeaderRegexp.FindAllStringSubmatch(string(diff), -1)

💯 agree. Felt this pain with gh agent as well.

### @BagToad reviewed on 2026-03-05 16:54

#### @BagToad commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {

+1 for excludePatterns

### @BagToad reviewed on 2026-03-05 16:56

#### @BagToad commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, patterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	var sections []string
+	for {
+		idx := strings.Index(diff, marker)
+		if idx == -1 {
+			if len(diff) > 0 {
+				sections = append(sections, diff)
+			}
+			break
+		}
+		sections = append(sections, diff[:idx+1]) // include the trailing \n
+		diff = diff[idx+1:]                       // next section starts at "diff --git"
+	}
+	return sections
+}

Those options are definitely cleaner. IDK if we need the regex; if we don't need the flexibility might be good to stick
with strings.SplitSeq

### @BagToad requested changes on 2026-03-05 17:40

Please re-request review when @babakks comments addressed 🙏 thank you

### @yuvrajangadsingh reviewed on 2026-03-06 08:09

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -92,6 +95,7 @@ func NewCmdDiff(f *cmdutil.Factory, runF func(*DiffOptions) error) *cobra.Comman
 	cmd.Flags().BoolVar(&opts.Patch, "patch", false, "Display diff in patch format")
 	cmd.Flags().BoolVar(&opts.NameOnly, "name-only", false, "Display only names of changed files")
 	cmd.Flags().BoolVarP(&opts.BrowserMode, "web", "w", false, "Open the pull request diff in the browser")
+	cmd.Flags().StringSliceVarP(&opts.Exclude, "exclude", "e", nil, "Exclude files matching glob `patterns` from the
diff")

done, added examples in 43c845e

### @yuvrajangadsingh reviewed on 2026-03-06 08:09

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)

already had this from the previous review round, confirmed it matches your suggestion

### @yuvrajangadsingh reviewed on 2026-03-06 08:09

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -292,8 +303,7 @@ func changedFilesNames(w io.Writer, r io.Reader) error {
 	// `"`` + hello-\360\237\230\200-world"
 	//
 	// Where I'm using the `` to indicate a string to avoid confusion with the " character.
-	pattern := regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
-	matches := pattern.FindAllStringSubmatch(string(diff), -1)
+	matches := diffHeaderRegexp.FindAllStringSubmatch(string(diff), -1)

agreed, would be nice to have. out of scope for this PR though

### @yuvrajangadsingh reviewed on 2026-03-06 08:09

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {

renamed to excludePatterns in 43c845e

### @yuvrajangadsingh reviewed on 2026-03-06 08:09

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, patterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	var sections []string
+	for {
+		idx := strings.Index(diff, marker)
+		if idx == -1 {
+			if len(diff) > 0 {
+				sections = append(sections, diff)
+			}
+			break
+		}
+		sections = append(sections, diff[:idx+1]) // include the trailing \n
+		diff = diff[idx+1:]                       // next section starts at "diff --git"
+	}
+	return sections
+}

switched to strings.Split in 43c845e, cleaner than the manual index approach

### @yuvrajangadsingh reviewed on 2026-03-06 08:10

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +367,65 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, patterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, patterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	var sections []string
+	for {
+		idx := strings.Index(diff, marker)
+		if idx == -1 {
+			if len(diff) > 0 {
+				sections = append(sections, diff)
+			}
+			break
+		}
+		sections = append(sections, diff[:idx+1]) // include the trailing \n
+		diff = diff[idx+1:]                       // next section starts at "diff --git"
+	}
+	return sections
+}
+
+func extractFileName(section []byte) string {
+	m := diffHeaderRegexp.FindSubmatch(section)
+	if m == nil {
+		return ""
+	}
+	return strings.TrimSpace(string(m[1]) + string(m[2]))
+}
+
+func matchesAny(name string, patterns []string) bool {
+	for _, p := range patterns {
+		if matched, _ := filepath.Match(p, name); matched {
+			return true
+		}
+		// Also match against the basename so "*.yml" matches "dir/file.yml"
+		if matched, _ := filepath.Match(p, filepath.Base(name)); matched {
+			return true
+		}
+	}
+	return false
+}

good call, switched from filepath.Match to path.Match in 43c845e. also added a note in the --help text that patterns use
forward slashes on all platforms.

### @yuvrajangadsingh commented on 2026-03-06 08:10

@babakks @BagToad addressed all feedback in 43c845e:

• added usage examples to --help
• switched filepath.Match to path.Match for OS-consistent behavior
• renamed patterns to excludePatterns
• simplified splitDiffSections with strings.Split

ready for another look when you get a chance.

### @babakks reviewed on 2026-03-09 12:54

Just a nitpick, @yuvrajangadsingh.

#### @babakks commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +384,67 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s("?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, excludePatterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, excludePatterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	parts := strings.Split(diff, marker)
+	if len(parts) == 1 {
+		return []string{diff}
+	}
+	sections := make([]string, 0, len(parts))
+	for i, p := range parts {
+		if i == 0 {
+			if len(p) > 0 {
+				sections = append(sections, p+"\n")
+			}
+		} else {
+			sections = append(sections, "diff --git "+p)
+		}
+	}
+	return sections
+}
+
+func extractFileName(section []byte) string {
+	m := diffHeaderRegexp.FindSubmatch(section)
+	if m == nil {
+		return ""
+	}
+	return strings.TrimSpace(string(m[1]) + string(m[2]))
+}

**nitpick:** can we change the signature from []byte to string here to avoid unnecessary conversion?

### @yuvrajangadsingh reviewed on 2026-03-09 14:02

#### @yuvrajangadsingh commented on pkg/cmd/pr/diff/diff.go

@@ -357,3 +384,67 @@ func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err e
 func isPrint(r rune) bool {
 	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
 }
+
+var diffHeaderRegexp = regexp.MustCompile(`(?:^|\n)diff\s--git.*\s("?)b/(.*)`)
+
+// filterDiff reads a unified diff and returns a new reader with file entries
+// matching any of the exclude patterns removed.
+func filterDiff(r io.Reader, excludePatterns []string) (io.Reader, error) {
+	data, err := io.ReadAll(r)
+	if err != nil {
+		return nil, err
+	}
+
+	var result bytes.Buffer
+	for _, section := range splitDiffSections(string(data)) {
+		name := extractFileName([]byte(section))
+		if name != "" && matchesAny(name, excludePatterns) {
+			continue
+		}
+		result.WriteString(section)
+	}
+	return &result, nil
+}
+
+// splitDiffSections splits a unified diff string into per-file sections.
+// Each section starts with "diff --git" and includes all content up to (but
+// not including) the next "diff --git" line.
+func splitDiffSections(diff string) []string {
+	marker := "\ndiff --git "
+	parts := strings.Split(diff, marker)
+	if len(parts) == 1 {
+		return []string{diff}
+	}
+	sections := make([]string, 0, len(parts))
+	for i, p := range parts {
+		if i == 0 {
+			if len(p) > 0 {
+				sections = append(sections, p+"\n")
+			}
+		} else {
+			sections = append(sections, "diff --git "+p)
+		}
+	}
+	return sections
+}
+
+func extractFileName(section []byte) string {
+	m := diffHeaderRegexp.FindSubmatch(section)
+	if m == nil {
+		return ""
+	}
+	return strings.TrimSpace(string(m[1]) + string(m[2]))
+}

done, switched to string param and used FindStringSubmatch in ea83ca0

### @babakks approved on 2026-03-09 15:01

LGTM! Thanks for your time and effort, @yuvrajangadsingh! 🙏 🚀

Tested on Linux and Windows, and works like a charm.

### @BagToad approved on 2026-03-09 15:07

