/** @type {import('tailwindcss').Config} */
module.exports = {
  // Paths to all of the template files in the project
  content: [
    "./pages/**/*.{js,ts,jsx,tsx}",
    "./components/**/*.{js,ts,jsx,tsx}",
    "./app/**/*.{js,ts,jsx,tsx}",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    // Extend the default Tailwind theme with our design‑system values
    extend: {
      colors: {
        primary: "#0F62FE",
        secondary: "#6F6F6F",
        success: "#24A148",
        danger: "#DA1E28",
        background: "#F4F7FA",
        card: "#FFFFFF",
        border: "#E0E0E0",
        // Provide a neutral gray scale derived from the secondary colour
        gray: {
          50: "#F9F9F9",
          100: "#F2F2F2",
          200: "#E5E5E5",
          300: "#D1D1D1",
          400: "#A6A6A6",
          500: "#6F6F6F",
          600: "#5A5A5A",
          700: "#404040",
          800: "#262626",
          900: "#0D0D0D",
        },
      },
      spacing: {
        // 4‑px grid system (base 1rem = 16px)
        0: "0rem",
        1: "0.25rem", // 4px
        2: "0.5rem", // 8px
        3: "0.75rem", // 12px
        4: "1rem", // 16px
        5: "1.25rem", // 20px (optional)
        6: "1.5rem", // 24px
        7: "1.75rem", // 28px (optional)
        8: "2rem", // 32px
        9: "2.25rem", // 36px (optional)
        10: "2.5rem", // 40px (optional)
        12: "3rem", // 48px
        14: "3.5rem", // 56px (optional)
        16: "4rem", // 64px (optional)
      },
      fontFamily: {
        // Inter is the primary typeface for the product
        sans: ["Inter", "system-ui", "sans-serif"],
      },
      fontSize: {
        // Base body text – 14px (0.875rem) with 1.5 line‑height
        base: ["0.875rem", { lineHeight: "1.5", letterSpacing: "-0.01em" }],
        // Heading scale – can be used via `text-xl`, `text-2xl`, etc.
        sm: ["0.9375rem", { lineHeight: "1.5" }], // 15px
        lg: ["1rem", { lineHeight: "1.5" }], // 16px
        xl: ["1.125rem", { lineHeight: "1.5" }], // 18px
        "2xl": ["1.25rem", { lineHeight: "1.5" }], // 20px
        "3xl": ["1.5rem", { lineHeight: "1.5" }], // 24px
        "4xl": ["1.875rem", { lineHeight: "1.5" }], // 30px
        "5xl": ["2.25rem", { lineHeight: "1.5" }], // 36px
      },
      borderRadius: {
        // Consistent rounded corners across components
        none: "0",
        sm: "0.125rem", // 2px
        DEFAULT: "0.5rem", // 8px – primary radius
        md: "0.75rem", // 12px
        lg: "1rem", // 16px
        full: "9999px",
      },
      boxShadow: {
        // Subtle elevation for cards, modals, and interactive elements
        sm: "0 1px 2px 0 rgba(0,0,0,0.05)",
        DEFAULT:
          "0 1px 3px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06)",
        md:
          "0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px rgba(0,0,0,0.06)",
        lg:
          "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px rgba(0,0,0,0.05)",
        xl:
          "0 20px 25px -5px rgba(0,0,0,0.1), 0 10px 10px rgba(0,0,0,0.04)",
        "2xl": "0 25px 50px -12px rgba(0,0,0,0.25)",
        inner: "inset 0 2px 4px 0 rgba(0,0,0,0.06)",
        none: "none",
      },
      // Focus ring that respects the design system and accessibility requirements
      ringColor: {
        DEFAULT: "#0F62FE",
      },
      ringWidth: {
        DEFAULT: "2px",
      },
    },
    // Center the container and give it a sensible max‑width
    container: {
      center: true,
      padding: {
        DEFAULT: "1rem",
        sm: "1.5rem",
        lg: "2rem",
        xl: "3rem",
      },
    },
  },
  // Enable additional utilities that are useful for forms and rich text
  plugins: [
    require("@tailwindcss/forms"),
    require("@tailwindcss/typography"),
  ],
  // Ensure the generated CSS is as small as possible in production
  corePlugins: {
    // Keep the preflight base styles – they are required for consistent resets
    preflight: true,
  },
};