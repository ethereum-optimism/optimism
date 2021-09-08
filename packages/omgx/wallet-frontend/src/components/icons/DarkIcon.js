import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function DarkIcon() {
  const theme = useTheme();
  const isLight = theme.palette.mode === 'light';
  const color = theme.palette.common[isLight ? 'black' : 'white'];
  return (
    <svg
      width={32}
      height={32}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <mask id="dark-icon" fill="#fff">
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M12 19a7 7 0 005.897-10.773 8 8 0 11-9.67 9.67A6.967 6.967 0 0012 19z"
        />
      </mask>
      <path
        d="M17.897 8.227l.354-1.458-3.639-.885 2.022 3.152 1.263-.81zm-9.67 9.67l.809-1.263-3.152-2.021.885 3.638 1.457-.354zM17.5 12a5.5 5.5 0 01-5.5 5.5v3a8.5 8.5 0 008.5-8.5h-3zm-.866-2.964c.548.855.866 1.87.866 2.964h3c0-1.685-.492-3.26-1.34-4.583l-2.526 1.62zm.908.648A6.503 6.503 0 0122.5 16h3c0-4.472-3.09-8.22-7.25-9.231l-.708 2.915zM22.5 16a6.5 6.5 0 01-6.5 6.5v3a9.5 9.5 0 009.5-9.5h-3zM16 22.5a6.503 6.503 0 01-6.316-4.957l-2.915.708c1.01 4.16 4.758 7.25 9.23 7.25v-3zm-4-5a5.468 5.468 0 01-2.964-.866l-1.62 2.526A8.467 8.467 0 0012 20.5v-3z"
        fill={color}
        mask="url(#dark-icon)"
      />
    </svg>
  )
}

export default DarkIcon;
