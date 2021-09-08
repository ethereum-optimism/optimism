/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */

import { lazy } from "react"


// TODO: move me to the proper folder.

// FIXME: MOVE ALL THE COMPONENTS FROM components to pages folder.
const WalletPage = lazy(()=> import('../pages/wallet/index'))
const FarmPage = lazy(()=> import('../pages/farm/index'))
const LearnPage = lazy(()=> import('../pages/learn/index'))
const PoolPage = lazy(()=> import('../pages/pool/index'))
const HistoryPage = lazy(()=> import('../pages/history/index'))


export const routeConfig = [
    {
        path: '/',
        exact: true,
        component: WalletPage
    },
    {
        path: '/pool',
        exact: true,
        component: PoolPage,
    },
    {
        path: '/farm',
        exact: true,
        component: FarmPage
    },
    {
        path: '/history',
        exact: true,
        component: HistoryPage
    },
    {
        path: '/learn',
        exact: true,
        component: LearnPage
    }
]