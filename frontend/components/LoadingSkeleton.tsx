import React from 'react';
import classNames from 'classnames';

/**
 * LoadingSkeleton – reusable placeholder component used while data is being fetched.
 *
 * Variants:
 *  - `text`   : three lines of varying width, mimicking paragraph text.
 *  - `title`  : single larger line, suitable for headings.
 *  - `avatar` : circular placeholder, useful for profile pictures.
 *  - `card`   : rectangular block, ideal for cards or panels.
 *  - `button` : rounded rectangle, matching button height.
 *  - `custom` : user‑defined dimensions via `width` / `height`.
 *
 * All skeletons use Tailwind's `animate-pulse` for the loading shimmer,
 * respect the design system's neutral colors, and are hidden from assistive
 * technologies (`aria-hidden="true"`).
 */
export type LoadingSkeletonProps = {
  /** Visual style of the skeleton */
  variant?: 'text' | 'title' | 'avatar' | 'card' | 'button' | 'custom';
  /** Width of the skeleton (Tailwind class like `w-32` or CSS value like `200px`) */
  width?: string;
  /** Height of the skeleton (Tailwind class like `h-8` or CSS value like `2rem`) */
  height?: string;
  /** Additional Tailwind or custom classes */
  className?: string;
};

/**
 * Helper to convert a Tailwind spacing/value into a class string.
 * If the value already looks like a Tailwind class (contains `-`), we return it unchanged.
 * Otherwise we treat it as a raw CSS value and apply it via inline style.
 */
const resolveSize = (
  size?: string,
  defaultClass?: string,
  styleObj?: React.CSSProperties,
) => {
  if (!size) return { class: defaultClass, style: styleObj };
  // Tailwind utility classes contain a dash (e.g., w-64, h-12)
  if (size.includes('-')) {
    return { class: size, style: styleObj };
  }
  // Raw CSS value – apply via inline style
  return {
    class: defaultClass,
    style: { ...styleObj, ...(size ? { width: size, height: size } : {}) },
  };
};

export const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  variant = 'custom',
  width,
  height,
  className = '',
}) => {
  // Base skeleton styling – neutral background, pulse animation, rounded corners
  const baseClasses =
    'bg-gray-200 dark:bg-gray-700 animate-pulse ease-in-out transition-opacity';

  // Resolve size classes / inline styles
  const size = resolveSize(width, 'w-full', {});
  const heightSize = resolveSize(height, undefined, {});

  // Build the final class string based on variant
  const skeletonClass = classNames(
    baseClasses,
    className,
    {
      // Text lines – multiple rows with varying widths
      'space-y-2': variant === 'text',
      // Title – a single, slightly taller line
      'h-6': variant === 'title',
      // Avatar – circle
      'rounded-full': variant === 'avatar',
      // Card – rounded rectangle
      'rounded-md': variant === 'card' || variant === 'button',
      // Button – more rounded and fixed height
      'h-10 rounded-md': variant === 'button',
    },
    // Apply explicit width/height if supplied
    size.class,
    heightSize.class,
  );

  // Inline style overrides (for raw CSS values)
  const inlineStyle: React.CSSProperties = {
    ...size.style,
    ...heightSize.style,
  };

  // Render based on variant
  if (variant === 'text') {
    // Three lines with decreasing width to simulate paragraph text
    return (
      <div aria-hidden="true" className={classNames('space-y-2', className)}>
        <div
          className={classNames(baseClasses, 'h-4 w-3/4 rounded')}
          style={inlineStyle}
        />
        <div
          className={classNames(baseClasses, 'h-4 w-5/6 rounded')}
          style={inlineStyle}
        />
        <div
          className={classNames(baseClasses, 'h-4 w-2/3 rounded')}
          style={inlineStyle}
        />
      </div>
    );
  }

  // Avatar – enforce equal width/height if only one dimension is provided
  if (variant === 'avatar') {
    const avatarSize = resolveSize(width || height, 'w-12 h-12', {});
    return (
      <div
        aria-hidden="true"
        className={classNames(baseClasses, 'rounded-full', className, avatarSize.class)}
        style={avatarSize.style}
      />
    );
  }

  // Card – default to full width with a minimum height if none supplied
  if (variant === 'card') {
    const cardHeight = height || 'h-48';
    const cardSize = resolveSize(width, 'w-full', {});
    return (
      <div
        aria-hidden="true"
        className={classNames(
          baseClasses,
          'rounded-md',
          cardSize.class,
          cardHeight,
          className,
        )}
        style={cardSize.style}
      />
    );
  }

  // Button – fixed height, width can be customized
  if (variant === 'button') {
    const btnSize = resolveSize(width, 'w-32', {});
    return (
      <div
        aria-hidden="true"
        className={classNames(
          baseClasses,
          'h-10 rounded-md',
          btnSize.class,
          className,
        )}
        style={btnSize.style}
      />
    );
  }

  // Title – single line
  if (variant === 'title') {
    const titleSize = resolveSize(width, 'w-1/2', {});
    return (
      <div
        aria-hidden="true"
        className={classNames(baseClasses, 'h-6 rounded', titleSize.class, className)}
        style={titleSize.style}
      />
    );
  }

  // Custom – fallback to generic rectangle
  return (
    <div
      aria-hidden="true"
      className={classNames(skeletonClass, className)}
      style={inlineStyle}
    />
  );
};

export default LoadingSkeleton;