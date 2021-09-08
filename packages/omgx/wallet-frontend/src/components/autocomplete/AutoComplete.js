import React from 'react';
import Autocomplete from '@material-ui/core/Autocomplete';
import { isEqual } from 'lodash';

class AutoComplete extends React.Component {

  constructor(props) {

    super(props);

    this.state = {
      placeholder: this.props.placeholder || '',
      selectionList: this.props.selectionList || '',
      excludeItem: this.props.excludeItem || '',
      excludeList: this.props.excludeList || [],
      onBlurDisappear: this.props.onBlurDisappear ? true : false,
      openControlLength: this.props.openControlLength || 0,
      passValue: this.props.passValue || {},
      selectedItem: '',
      openControl: false,
    }

  }

  componentDidUpdate(prevState) {
    if (!isEqual(prevState.selectionList, this.props.selectionList)) {
      this.setState({ selectionList: this.props.selectionList });
    }

    if (prevState.excludeItem !== this.props.excludeItem) {
      this.setState({ excludeItem: this.props.excludeItem });
    }

    if (prevState.excludeList !== this.props.excludeList) {
      this.setState({ excludeList: this.props.excludeList });
    }

    if (prevState.passValue !== this.props.passValue) {
      this.setState({ passValue: this.props.passValue });
      this.setState({ selectedItem: this.props.passValue.symbol });
    }
  }

  handleUpdateValue(event, value, reason) {
    this.props.updateValue(value);
  }

  handleInputValue(value, reason) {
    const { openControlLength, onBlurDisappear } = this.state;

    if (value.length >= openControlLength && openControlLength && reason === 'input') {
      this.setState({ openControl: true });
    } else if (openControlLength) {
      this.setState({ openControl: false });
    }

    if (reason !== 'reset') {
      this.setState({ selectedItem: value })
    }

    if (reason === 'reset') {
      if (value) {
        if (onBlurDisappear) {
          this.setState({ selectedItem: '' });
        } else {
          this.setState({ selectedItem: value });
        }
      }
    }
  }

  render() {

    const {
      selectedItem,
      placeholder,
      selectionList,
      // exclude
      excludeItem,
      excludeList,
      // blur
      onBlurDisappear,
      // open control
      openControl,
      openControlLength,
    } = this.state;

    let options = [];
    Object.values(selectionList).forEach(val => {
      if (val.symbol !== excludeItem &&
        !excludeList.includes(val.symbol) &&
        val.symbol !== 'Not found' &&
        val.symbol !== 'ETH'
      ) {
        options.push(val.symbol);
      }
    });

    let AutocompleteAttribute = {}
    if (openControlLength) {
      AutocompleteAttribute.open = openControl;
    }

    return (
      <Autocomplete
        {...AutocompleteAttribute}
        inputValue={selectedItem}
        clearOnBlur={true}
        blurOnSelect={'mouse'}
        options={options}
        getOptionSelected={(option, value) => true}
        onChange={(event, value, reason)=>this.handleUpdateValue(event, value, reason)}
        onInputChange={(event, value, reason)=>{this.handleInputValue(value, reason)}}
        renderInput={(params) => (
          <div ref={params.InputProps.ref}>
            <input
              placeholder={placeholder}
              type="text"
              {...params.inputProps}
              value={onBlurDisappear ?
                selectedItem ? params.inputProps.value : '':
                params.inputProps.value
              }
            >
            </input>
          </div>
        )}
      />
    )
  }
}

export default AutoComplete;
