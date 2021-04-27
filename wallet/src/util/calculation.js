/*
  Varna - A Privacy-Preserving Marketplace
  Varna uses Fully Homomorphic Encryption to make markets fair. 
  Copyright (C) 2021 Enya Inc. Palo Alto, CA

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

var accAdd = function(num1, num2) {
    num1 = Number(num1);
    num2 = Number(num2);
    var dec1, dec2, times;
    try { dec1 = countDecimals(num1)+1; } catch (e) { dec1 = 0; }
    try { dec2 = countDecimals(num2)+1; } catch (e) { dec2 = 0; }
    times = Math.pow(10, Math.max(dec1, dec2));
    var result = (accMul(num1, times) + accMul(num2, times)) / times;
    return getCorrectResult("add", num1, num2, result);
};

var accSub = function(num1, num2) {
    num1 = Number(num1);
    num2 = Number(num2);
    var dec1, dec2, times;
    try { dec1 = countDecimals(num1)+1; } catch (e) { dec1 = 0; }
    try { dec2 = countDecimals(num2)+1; } catch (e) { dec2 = 0; }
    times = Math.pow(10, Math.max(dec1, dec2));
    var result = Number((accMul(num1, times) - accMul(num2, times)) / times);
    return getCorrectResult("sub", num1, num2, result);
};

var accDiv = function(num1, num2) {
    num1 = Number(num1);
    num2 = Number(num2);
    var t1 = 0, t2 = 0, dec1, dec2;
    try { t1 = countDecimals(num1); } catch (e) { }
    try { t2 = countDecimals(num2); } catch (e) { }
    dec1 = convertToInt(num1);
    dec2 = convertToInt(num2);
    var result = accMul((dec1 / dec2), Math.pow(10, t2 - t1));
    return getCorrectResult("div", num1, num2, result);
};

var accMul = function(num1, num2) {
    num1 = Number(num1);
    num2 = Number(num2);
    var times = 0, s1 = num1.toString(), s2 = num2.toString();
    try { times += countDecimals(s1); } catch (e) { }
    try { times += countDecimals(s2); } catch (e) { }
    var result = convertToInt(s1) * convertToInt(s2) / Math.pow(10, times);
    return getCorrectResult("mul", num1, num2, result);
};

var countDecimals = function(num) {
    var len = 0;
    try {
        num = Number(num);
        var str = num.toString().toUpperCase();
        if (str.split('E').length === 2) { // scientific notation
            var isDecimal = false;
            if (str.split('.').length === 2) {
                str = str.split('.')[1];
                if (parseInt(str.split('E')[0]) !== 0) {
                    isDecimal = true;
                }
            }
            let x = str.split('E');
            if (isDecimal) {
                len = x[0].length;
            }
            len -= parseInt(x[1]);
        } else if (str.split('.').length === 2) { // decimal
            if (parseInt(str.split('.')[1]) !== 0) {
                len = str.split('.')[1].length;
            }
        }
    } catch(e) {
        throw e;
    } finally {
        if (isNaN(len) || len < 0) {
            len = 0;
        }
        return len;
    }
};

var convertToInt = function(num) {
    num = Number(num);
    var newNum = num;
    var times = countDecimals(num);
    var temp_num = num.toString().toUpperCase();
    if (temp_num.split('E').length === 2) {
        newNum = Math.round(num * Math.pow(10, times));
    } else {
        newNum = Number(temp_num.replace(".", ""));
    }
    return newNum;
};

var getCorrectResult = function(type, num1, num2, result) {
    var temp_result = 0;
    switch (type) {
        case "add":
            temp_result = num1 + num2;
            break;
        case "sub":
            temp_result = num1 - num2;
            break;
        case "div":
            temp_result = num1 / num2;
            break;
        case "mul":
            temp_result = num1 * num2;
            break;
        default:
            break;
    }
    if (Math.abs(result - temp_result) > 1) {
        return temp_result;
    }
    return result;
};


export {
  accAdd,
  accSub,
  accDiv,
  accMul,
  countDecimals,
  convertToInt,
  getCorrectResult,
}