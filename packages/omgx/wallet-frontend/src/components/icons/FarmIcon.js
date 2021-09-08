import * as React from "react"

function FarmIcon({ color }) {
  return (
    <svg
      width={18}
      height={19}
      viewBox="0 0 18 19"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M.75 1.5h7.5a2.25 2.25 0 012.25 2.25v3.557H.75V1.5z"
        stroke={color}
        strokeWidth={1.5}
      />
      <path
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        d="M15 6.75V2.625"
      />
      <circle
        cx={3.938}
        cy={14.813}
        r={3.188}
        stroke={color}
        strokeWidth={1.5}
      />
      <circle
        cx={14.625}
        cy={15.375}
        r={2.625}
        stroke={color}
        strokeWidth={1.5}
      />
      <mask id="prefix__a" fill="#fff">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M0 6.375h15a3 3 0 013 3v6a3.375 3.375 0 11-6.75 0H7.835a3.938 3.938 0 01-7.795 0H0v-9z"
        />
      </mask>
      <path
        d="M0 6.375v-1.5h-1.5v1.5H0zm11.25 9h1.5v-1.5h-1.5v1.5zm-3.415 0v-1.5h-1.3l-.185 1.288 1.485.212zm-7.795 0l1.485-.212-.184-1.288H.04v1.5zm-.04 0h-1.5v1.5H0v-1.5zm15-10.5H0v3h15v-3zm4.5 4.5a4.5 4.5 0 00-4.5-4.5v3a1.5 1.5 0 011.5 1.5h3zm0 6v-6h-3v6h3zm-3 0c0 1.035-.84 1.875-1.875 1.875v3a4.875 4.875 0 004.875-4.875h-3zm-1.875 1.875a1.875 1.875 0 01-1.875-1.875h-3a4.875 4.875 0 004.875 4.875v-3zm-6.79-.375h3.415v-3H7.835v3zM3.938 20.25a5.438 5.438 0 005.382-4.663l-2.97-.424a2.438 2.438 0 01-2.412 2.087v3zm-5.383-4.663a5.438 5.438 0 005.383 4.663v-3a2.438 2.438 0 01-2.413-2.087l-2.97.424zM0 16.875h.04v-3H0v3zm-1.5-2.063v.563h3v-.563h-3zm0-8.437v8.438h3V6.374h-3z"
        fill={color}
        mask="url(#prefix__a)"
      />
    </svg>
  )
}

export default FarmIcon
