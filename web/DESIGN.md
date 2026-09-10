---
version: alpha
name: Altermind
description: A dark, editorial brand system that blends academic prestige with restrained modernism.
colors:
  primary: "#001F1E"
  secondary: "#0B3B39"
  tertiary: "#374151"
  neutral: "#0B1D1C"
  surface: "#001F1E"
  on-surface: "#FFFFFF"
  accent: "#FFFFFF"
  border: "#FFFFFF38"
  error: "#D34B4B"
typography:
  headline-display:
    fontFamily: Freight
    fontSize: 168px
    fontWeight: 400
    lineHeight: 202px
    letterSpacing: 0px
  headline-xl:
    fontFamily: Freight
    fontSize: 101px
    fontWeight: 400
    lineHeight: 121px
    letterSpacing: 0px
  headline-lg:
    fontFamily: Freight
    fontSize: 60px
    fontWeight: 400
    lineHeight: 72px
    letterSpacing: 0px
  headline-md:
    fontFamily: Freight
    fontSize: 36px
    fontWeight: 400
    lineHeight: 43px
    letterSpacing: 0px
  body-lg:
    fontFamily: "Basis Grotesque Pro"
    fontSize: 21.6px
    fontWeight: 400
    lineHeight: 32px
    letterSpacing: 1.2px
  body-md:
    fontFamily: "Basis Grotesque Pro"
    fontSize: 16.8px
    fontWeight: 400
    lineHeight: 24px
    letterSpacing: 0.8px
  body-sm:
    fontFamily: "Basis Grotesque Pro"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 20px
    letterSpacing: 0.6px
  label-lg:
    fontFamily: "Basis Grotesque Pro"
    fontSize: 16px
    fontWeight: 400
    lineHeight: 24px
    letterSpacing: 0.4px
  label-md:
    fontFamily: "Basis Grotesque Pro"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 20px
    letterSpacing: 0.4px
  label-sm:
    fontFamily: "Basis Grotesque Pro"
    fontSize: 12px
    fontWeight: 500
    lineHeight: 16px
    letterSpacing: 0.08em
rounded:
  none: 0px
  sm: 4px
  md: 8px
  lg: 16px
  xl: 24px
  full: 9999px
spacing:
  xs: 14px
  sm: 36px
  md: 56px
  lg: 108px
  xl: 168px
components:
  button-primary:
    backgroundColor: "transparent"
    textColor: "{colors.on-surface}"
    typography: "{typography.body-md}"
    rounded: "{rounded.full}"
    padding: "19px 28px"
    height: "55px"
  button-primary-hover:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-surface}"
    typography: "{typography.body-md}"
    rounded: "{rounded.full}"
    padding: "19px 28px"
    height: "55px"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.on-surface}"
    typography: "{typography.body-md}"
    rounded: "{rounded.full}"
    padding: "19px 28px"
    height: "55px"
  button-link:
    backgroundColor: "transparent"
    textColor: "{colors.on-surface}"
    typography: "{typography.label-md}"
    rounded: "{rounded.none}"
    padding: "0px"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.md}"
    padding: "16px"
  input:
    backgroundColor: "transparent"
    textColor: "{colors.on-surface}"
    typography: "{typography.body-md}"
    rounded: "{rounded.full}"
    padding: "16px 20px"
  chip:
    backgroundColor: "transparent"
    textColor: "{colors.on-surface}"
    typography: "{typography.label-sm}"
    rounded: "{rounded.full}"
    padding: "8px 12px"
---

# Altermind

## Overview
Altermind feels like a dark, cinematic editorial brand with an academic edge. The visual tone is elevated and intellectual, but still contemporary and restrained, making it suitable for consulting, research, innovation, or premium thought-leadership content. Spacious framing, minimal chrome, and high-contrast typography create a composed, confident presence rather than a playful one.

## Colors
- **Primary (#001F1E):** A deep blue-green black used as the base background and primary surface color. It establishes the brand’s moody, immersive atmosphere.
- **Secondary (#0B3B39):** A darker teal accent for subtle tonal shifts, hover states, and layered depth in the background language.
- **Tertiary (#374151):** A muted graphite tone for quiet borders and structural separation when a softer outline is needed.
- **Neutral (#0B1D1C):** A near-black supporting tone that can anchor overlays, inset areas, or secondary dark surfaces.
- **Surface (#001F1E):** The primary canvas color, matching the brand’s dark stage and keeping the UI visually continuous.
- **On-surface (#FFFFFF):** Pure white text and iconography used for maximum clarity and editorial contrast.
- **Accent (#FFFFFF):** The bright highlight color for logos, primary labels, buttons, and emphasis points.
- **Border (#FFFFFF38):** A translucent white border tint used for pill buttons, circular controls, and subtle outlines.
- **Error (#D34B4B):** A restrained red reserved for validation or alert states so it does not disrupt the calm palette.

## Typography
Freight provides the distinctive editorial voice for display headings. It is used in large, elegant, serif compositions with generous scale and tight visual discipline, especially for the hero wordmark-style title. Basis Grotesque Pro handles all supporting text, delivering a clean sans-serif counterbalance for navigation, body copy, and labels.

Headlines are light in weight at 400 and rely on scale rather than boldness. The largest titles are dramatic and spacious, while smaller headings remain refined and measured. Body text uses slightly expanded letter spacing, echoing the screenshot’s uppercase, tracked navigation and CTA copy. Labels and navigation should stay in uppercase or small-caps-like treatment where appropriate, with moderate tracking for the polished academic feel.

## Layout
The layout is highly centered and theatrical, with a hero-first composition and large surrounding negative space. Navigation is distributed across the top edge, while the main title and CTA sit in the center/bottom-center axis to create a clear vertical hierarchy. The spacing rhythm is broad and deliberate, using large jumps rather than dense increments, matching the `xs` through `xl` scale.

This system favors a fluid full-bleed canvas over boxed containers. When content is grouped, use generous inset padding and preserve wide margins so the interface feels expansive and premium. Cards and panels should remain minimal, with modest internal padding and plenty of breathing room around them.

## Elevation & Depth
The brand is intentionally flat in its UI components: shadows are absent, and depth comes from tonal layering, gradients, and overlapping diagonal forms in the background. Hierarchy is created through contrast, translucency, and edge definition rather than raised surfaces. Use subtle borders like `border` and `tertiary` to separate elements when needed, but avoid heavy drop shadows or glossy effects.

## Shapes
The shape language is soft and architectural. Interactive controls are predominantly pill-shaped, using `rounded.full` for buttons, chips, and small utility controls. Secondary containers use `rounded.md` for a controlled 8px radius, which keeps the interface refined without feeling overly rounded. Overall, the system balances strict geometry with gentle curves.

## Components
Buttons are understated and elegant. Primary and secondary buttons should use transparent backgrounds, white text, thin translucent borders, and pill radii. The main CTA should feel spacious, with `19px 28px` padding and a minimum height around `55px`, matching the observed centered button. Hover states can subtly shift toward `secondary` while preserving the transparent, outlined character.

Links and navigation items should be typographic rather than button-like. Use Basis Grotesque Pro with light-to-regular weight, uppercase styling where needed, and no decorative underlines unless active or hovered. The navigation bar should feel airy, evenly spaced, and aligned to the top edge.

Cards should be quiet, dark panels with a fine border and minimal or no shadow. Keep padding modest, as in the observed `16px` card spacing, and let contrast, spacing, and typography carry the hierarchy. Inputs should follow the same pill-based outline language as buttons, with transparent fills, white text, and restrained borders.

Chips and small utility controls should remain compact and circular or pill-shaped. Use the smallest label styling with modest tracking so they read as interface signals rather than decorative tags. Icons should be monochrome white, thin, and restrained to match the overall precision of the system.

## Do's and Don'ts
- Do keep everything high-contrast, dark, and refined.
- Do use Freight only for headline or hero-scale moments.
- Do rely on spacing and typography for hierarchy instead of shadows.
- Do keep buttons and controls pill-shaped with thin translucent borders.
- Don't introduce bright, saturated colors that break the calm academic mood.
- Don't use heavy shadows, skeuomorphic effects, or glossy gradients.
- Don't make body copy dense; preserve wide line lengths and generous breathing room.
- Don't mix in playful rounded corners or noisy decorative UI treatments.
