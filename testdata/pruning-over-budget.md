# Pruning Over Budget
<!-- fixture: exercises all three PruneNavEntries branches under default config -->
<!-- Large section (N>=150): h3 children are unkillable -->
<!-- Medium section (50<=N<150): h3 children are hintable -->
<!-- Small section (N<50): h3 children are droppable -->

This file is a static test fixture. It is not a real document.
It is designed so that running `agentmap generate` (or `agentmap index`) on it
produces a nav block that clearly shows all three pruning branches:
unkillable (large), hintable (medium), and droppable (small).

Do not edit line counts without recalculating section sizes.
Total headings: 1 h1 + 9 h2 + 25 h3 = 35 entries, budget 20.

---

## Filler One

Short section with no subsections.
Used to inflate entry count without affecting pruning classification.
Content line four.
Content line five.

## Filler Two

Short section with no subsections.
Used to inflate entry count without affecting pruning classification.
Content line four.
Content line five.

## Filler Three

Short section with no subsections.
Used to inflate entry count without affecting pruning classification.
Content line four.
Content line five.

## Filler Four

Short section with no subsections.
Used to inflate entry count without affecting pruning classification.
Content line four.
Content line five.

## Filler Five

Short section with no subsections.
Used to inflate entry count without affecting pruning classification.
Content line four.
Content line five.

## Filler Six

Short section with no subsections.
Used to inflate entry count without affecting pruning classification.
Content line four.
Content line five.

## Large Section
<!-- target N >= 150; h3 children should be unkillable -->

This section is large enough that its h3 children are unkillable by the pruner.
A parent with N >= 150 (expand_threshold default) causes its children to be kept
even when the nav block is over budget. The pruner accepts the overrun rather than
removing entries from a large section.

Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.

### Large Alpha

Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.
Content for subsection Large Alpha.

### Large Beta

Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.
Content for subsection Large Beta.

### Large Gamma

Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.
Content for subsection Large Gamma.

### Large Delta

Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.
Content for subsection Large Delta.

### Large Epsilon

Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.
Content for subsection Large Epsilon.

### Large Zeta

Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.
Content for subsection Large Zeta.

### Large Eta

Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.
Content for subsection Large Eta.

### Large Theta

Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.
Content for subsection Large Theta.

### Large Iota

Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.
Content for subsection Large Iota.

## Medium Section
<!-- target 50 <= N < 150; h3 children should be hintable -->

This section is medium sized. Its h3 children are hintable: when the pruner
removes them to meet the budget it also appends their names as hints on this
parent's about field using the > prefix convention.

Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.
Line of filler content to pad the section size.

### Medium Alpha

Content for subsection Medium Alpha.
Content for subsection Medium Alpha.
Content for subsection Medium Alpha.
Content for subsection Medium Alpha.
Content for subsection Medium Alpha.
Content for subsection Medium Alpha.

### Medium Beta

Content for subsection Medium Beta.
Content for subsection Medium Beta.
Content for subsection Medium Beta.
Content for subsection Medium Beta.
Content for subsection Medium Beta.
Content for subsection Medium Beta.

### Medium Gamma

Content for subsection Medium Gamma.
Content for subsection Medium Gamma.
Content for subsection Medium Gamma.
Content for subsection Medium Gamma.
Content for subsection Medium Gamma.
Content for subsection Medium Gamma.

### Medium Delta

Content for subsection Medium Delta.
Content for subsection Medium Delta.
Content for subsection Medium Delta.
Content for subsection Medium Delta.
Content for subsection Medium Delta.
Content for subsection Medium Delta.

### Medium Epsilon

Content for subsection Medium Epsilon.
Content for subsection Medium Epsilon.
Content for subsection Medium Epsilon.
Content for subsection Medium Epsilon.
Content for subsection Medium Epsilon.
Content for subsection Medium Epsilon.

### Medium Zeta

Content for subsection Medium Zeta.
Content for subsection Medium Zeta.
Content for subsection Medium Zeta.
Content for subsection Medium Zeta.
Content for subsection Medium Zeta.
Content for subsection Medium Zeta.

### Medium Eta

Content for subsection Medium Eta.
Content for subsection Medium Eta.
Content for subsection Medium Eta.
Content for subsection Medium Eta.
Content for subsection Medium Eta.
Content for subsection Medium Eta.

### Medium Theta

Content for subsection Medium Theta.
Content for subsection Medium Theta.
Content for subsection Medium Theta.
Content for subsection Medium Theta.
Content for subsection Medium Theta.
Content for subsection Medium Theta.

## Small Section
<!-- target N < 50; h3 children should be droppable (no hint) -->

This section is small. Its h3 children are silently dropped.

### Small Alpha

Content for subsection Small Alpha.
Content for subsection Small Alpha.

### Small Beta

Content for subsection Small Beta.
Content for subsection Small Beta.

### Small Gamma

Content for subsection Small Gamma.
Content for subsection Small Gamma.

### Small Delta

Content for subsection Small Delta.
Content for subsection Small Delta.

### Small Epsilon

Content for subsection Small Epsilon.
Content for subsection Small Epsilon.

### Small Zeta

Content for subsection Small Zeta.
Content for subsection Small Zeta.

### Small Eta

Content for subsection Small Eta.
Content for subsection Small Eta.

### Small Theta

Content for subsection Small Theta.
Content for subsection Small Theta.
