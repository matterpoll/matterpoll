import React from 'react';
import PropTypes from 'prop-types';

import DefaultSetting from './default_setting';

export default class DefaultSettings extends React.Component {
    static propTypes = {
        id: PropTypes.string.isRequired,
        label: PropTypes.string.isRequired,
        helpText: PropTypes.node,
        value: PropTypes.any,
        disabled: PropTypes.bool.isRequired,
        config: PropTypes.object.isRequired,
        license: PropTypes.object.isRequired,
        setByEnv: PropTypes.bool.isRequired,
        onChange: PropTypes.func.isRequired,
        registerSaveAction: PropTypes.func.isRequired,
        setSaveNeeded: PropTypes.func.isRequired,
        unRegisterSaveAction: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        this.settings = {
            ...props.value,
        };
    }

    handleChange = (name, value) => {
        this.settings[name] = value;
        this.props.onChange(this.props.id, this.settings);
        this.props.setSaveNeeded();
    };

    render() {
        return (
            <div>
                <strong>{'Default settings'}</strong>

                <DefaultSetting
                    name={'anonymous'}
                    title={'Anonymous'}
                    label={'Don\'t show who voted for what when the poll ends'}
                    value={this.settings.anonymous}
                    onChange={this.handleChange}
                />
                <DefaultSetting
                    name={'anonymousCreator'}
                    title={'Anonymous Creator'}
                    label={'Don\'t show author of the poll'}
                    value={this.settings.anonymousCreator}
                    onChange={this.handleChange}
                />
                <DefaultSetting
                    name={'progress'}
                    title={'Progress'}
                    label={'During the poll, show how many votes each answer option got'}
                    value={this.settings.progress}
                    onChange={this.handleChange}
                />
                <DefaultSetting
                    name={'publicAddOption'}
                    title={'Public Add Option'}
                    label={'Allow all users to add additional options'}
                    value={this.settings.publicAddOption}
                    onChange={this.handleChange}
                />
            </div>
        );
    }
}
