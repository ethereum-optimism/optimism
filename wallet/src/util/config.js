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

import env from '@beam-australia/react-env';

export default {
  checkSyncInterval: env('SYNC_INTERVAL'),
  pollInterval: env('POLL_INTERVAL'),
  sentry: env('SENTRY_DSN'),
  gtmId: env('GTM_ID'),
  rpcProxy: env('RPC_PROXY')
};
