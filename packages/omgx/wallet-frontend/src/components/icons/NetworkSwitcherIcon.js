import { useTheme } from "@emotion/react";
import * as React from "react"

function NetworkSwitcherIcon({ active }) {
  const theme = useTheme();
  const isLight = theme.palette.mode === 'light';
  const color = !active ? '#506DFA' : isLight ? 'black' : '#506DFA';

  return (
    <svg
      width={26}
      height={26}
      viewBox="0 0 26 26"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle
        opacity={0.05}
        r={12.375}
        transform="matrix(1 0 0 -1 13 13)"
        fill="url(#prefix__paint0_linear)"
        fillOpacity={0.9}
      />
      <path
        opacity={0.24}
        d="M2.95 7.813c-.522-.27-1.17-.066-1.394.478a12.375 12.375 0 009.497 16.93c.582.093 1.093-.354 1.135-.941.043-.587-.4-1.092-.98-1.195a10.244 10.244 0 01-7.75-13.812c.215-.549.016-1.19-.507-1.46zM10.634 1.941c-.123-.576-.691-.947-1.254-.775a12.375 12.375 0 00-3.835 1.957c-.47.355-.503 1.033-.11 1.47.394.438 1.065.468 1.541.122A10.244 10.244 0 019.828 3.26c.56-.182.93-.743.806-1.319zM23.372 17.508c.54.235 1.172-.012 1.36-.57A12.374 12.374 0 0014.13.677c-.586-.054-1.067.425-1.07 1.014-.003.589.473 1.063 1.058 1.127a10.244 10.244 0 018.65 13.267c-.178.561.064 1.188.604 1.423z"
        fill={color}
      />
      <circle
        opacity={0.8}
        r={5.625}
        transform="matrix(1 0 0 -1 13 13)"
        fill={color}
      />
      <defs>
        <linearGradient
          id="prefix__paint0_linear"
          x1={6.497}
          y1={-5.869}
          x2={38.158}
          y2={22.453}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor={color} />
          <stop offset={1} stopColor={color} />
        </linearGradient>
      </defs>
    </svg>
  )
}

export default NetworkSwitcherIcon;
