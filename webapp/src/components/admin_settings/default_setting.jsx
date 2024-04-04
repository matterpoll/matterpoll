import React from 'react';
import PropTypes from 'prop-types';

export default class DefaultSetting extends React.Component {
    static propTypes = {
        name: PropTypes.string,
        title: PropTypes.string,
        label: PropTypes.string,
        value: PropTypes.bool,
        onChange: PropTypes.func.isRequired,
    };

    handleChange = (e) => {
        this.props.onChange(this.props.name, e.target.checked);
    };

    render() {
        return (
            <div
                className='row'
                style={styles.row}
            >
                <div
                    className='col-xs-12 col-sm-4'
                    style={styles.label}
                >
                    <strong>{this.props.title}</strong>
                </div>
                <div className='col-xs-12 col-sm-8'>
                    <div className='checkbox'>
                        <label>
                            <input
                                type='checkbox'
                                defaultChecked={this.props.value}
                                onClick={this.handleChange}
                            />
                            <span>{this.props.label}</span>
                        </label>
                    </div>
                </div>
            </div>
        );
    }
}

const styles = {
    row: {
        margin: '8px 0',
    },
    label: {
        marginTop: '4px',
    },
};
