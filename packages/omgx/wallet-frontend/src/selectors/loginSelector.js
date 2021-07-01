export function selectLogin () {
  return function (state) {
    return state.login.loggedIn;
  };
}
