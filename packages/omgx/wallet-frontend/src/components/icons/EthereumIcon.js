import * as React from "react"

function EthereumIcon({width, height}) {
  return (
    <svg
      width={width}
      height={height}
      viewBox="0 0 63 62"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect
        x={6}
        y={5}
        width={54}
        height={54}
        rx={27}
        fill="url(#prefix__paint0_radial)"
      />
      <g filter="url(#prefix__filter0_f)">
        <rect
          x={11}
          y={11}
          width={36}
          height={36}
          rx={18}
          fill="url(#prefix__paint1_radial)"
        />
      </g>
      <path
        opacity={0.6}
        d="M33.006 27.836l-9.846 4.47 9.846 5.806 9.843-5.806-9.843-4.47z"
        fill="#fff"
      />
      <path
        opacity={0.45}
        d="M23.16 32.306l9.846 5.806V16L23.16 32.306z"
        fill="#fff"
      />
      <path
        opacity={0.8}
        d="M33.006 16v22.112l9.843-5.806L33.006 16z"
        fill="#fff"
      />
      <path
        opacity={0.45}
        d="M23.16 34.168l9.846 13.845v-8.039l-9.846-5.806z"
        fill="#fff"
      />
      <path
        opacity={0.8}
        d="M33.006 39.974v8.04l9.85-13.846-9.85 5.806z"
        fill="#fff"
      />
      <defs>
        <radialGradient
          id="prefix__paint0_radial"
          cx={0}
          cy={0}
          r={1}
          gradientUnits="userSpaceOnUse"
          gradientTransform="rotate(-143.791 29.638 10.944) scale(40.7375)"
        >
          <stop stopColor="#0038FF" stopOpacity={0.67} />
          <stop offset={1} stopColor="#2B57F2" stopOpacity={0.44} />
        </radialGradient>
        <radialGradient
          id="prefix__paint1_radial"
          cx={0}
          cy={0}
          r={1}
          gradientUnits="userSpaceOnUse"
          gradientTransform="rotate(-143.791 24.512 9.985) scale(27.1583)"
        >
          <stop stopColor="#0038FF" stopOpacity={0.67} />
          <stop offset={1} stopColor="#2B57F2" stopOpacity={0.44} />
        </radialGradient>
        <filter
          id="prefix__filter0_f"
          x={0}
          y={0}
          width={58}
          height={58}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={5.5} result="effect1_foregroundBlur" />
        </filter>
      </defs>
    </svg>
  )
}

export default EthereumIcon
