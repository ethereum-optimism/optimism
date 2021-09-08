/* eslint-disable quotes */
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
import axios from 'axios';

const isIpfsUrl = (url) =>{
    return !!url.includes('ipfs://');
}

const getIpfsUrl = (url) => {
    let payload = url.split('://')[1]; 
    return `https://ipfs.io/ipfs/${payload}`
}


export const getNftImageUrl = async (url) => {
    try {

        let URL = !!isIpfsUrl(url) ? getIpfsUrl(url) : url;

        let res = await axios.get(URL);

        if (res.headers && res.headers['content-type'].includes('application/json')) {
            const { image, attributes = [] } = res.data;
            return { 
                url: !!isIpfsUrl(image) ? getIpfsUrl(image) : image, 
                attributes 
            }
        } else {
            return { url };
        }
    } catch (error) {
        // In case of error returning same url
        // As seems like some time it can be cors for images.
        console.log('Error while loading NFT image url', error.message);
        return { url };
    }
}


