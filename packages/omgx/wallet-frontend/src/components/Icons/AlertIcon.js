import * as React from "react";
import { useTheme } from "@material-ui/core/styles";

function AlertIcon() {
    const theme = useTheme();
    const isLight = theme.palette.mode === 'light';
    const color = theme.palette.common[isLight ? 'black' : 'white'];
    return (
        <svg width="26" height="24"
            viewBox="0 0 26 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg">
            <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M23.3922 17L15.598 3.5C14.4433 1.5 11.5566 1.5 10.4018 3.5L2.60762 17C1.45293 19 2.8963 21.5 5.20571 21.5H20.7942C23.1036 21.5 24.5469 19 23.3922 17ZM17.3301 2.5C15.4056 -0.833337 10.5943 -0.83333 8.6698 2.5L0.875573 16C-1.04893 19.3333 1.35671 23.5 5.20571 23.5H20.7942C24.6432 23.5 27.0488 19.3333 25.1243 16L17.3301 2.5ZM11.8 8.25847C11.8 7.79751 12.1737 7.42382 12.6346 7.42382C13.0956 7.42382 13.4693 7.79751 13.4693 8.25847V13.8969C13.4693 14.3578 13.0956 14.7315 12.6346 14.7315C12.1737 14.7315 11.8 14.3578 11.8 13.8969V8.25847ZM12.6347 16.1914C12.0293 16.1914 11.5385 16.6822 11.5385 17.2876C11.5385 17.8929 12.0293 18.3837 12.6347 18.3837C13.24 18.3837 13.7308 17.8929 13.7308 17.2876C13.7308 16.6822 13.24 16.1914 12.6347 16.1914Z"
                fill={color}
                fillOpacity="0.7" />
        </svg>
    );
}

export default AlertIcon;


