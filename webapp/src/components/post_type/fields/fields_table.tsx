import React from 'react';

import type {FormatTextOptions} from '@/types/mattermost-webapp';
import type {Attachment} from '@/types/poll';

const {formatText, messageHtmlToComponent} = window.PostUtils;

const halfWidth: React.CSSProperties = {width: '50%'};

type Props = {
    attachment: Attachment;
    options?: FormatTextOptions;
};

export default class FieldsTable extends React.PureComponent<Props> {
    render() {
        const fields = this.props.attachment.fields;
        if (!fields || !fields.length) {
            return '';
        }

        const fieldTables = [];

        let headerCols: React.ReactNode[] = [];
        let bodyCols: React.ReactNode[] = [];
        let rowPos = 0;
        let lastWasLong = false;
        let nrTables = 0;

        fields.forEach((field, i) => {
            // Nothing accumulated yet means there is no row to flush -- without this the
            // first field, if long, pushed an empty table ahead of itself.
            const shouldFlush = rowPos === 2 || !(field.short === true) || lastWasLong;
            if (shouldFlush && headerCols.length > 0) {
                fieldTables.push(
                    <table
                        className='attachment-fields'
                        key={'attachment__table__' + nrTables}
                    >
                        <thead>
                            <tr>
                                {headerCols}
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                {bodyCols}
                            </tr>
                        </tbody>
                    </table>,
                );
                headerCols = [];
                bodyCols = [];
                rowPos = 0;
                nrTables += 1;
                lastWasLong = false;
            }

            const fieldTitle = messageHtmlToComponent(formatText(field.title, this.props.options));
            headerCols.push(
                <th
                    className='attachment-field__caption'
                    key={'attachment__field-caption-' + i + '__' + nrTables}
                    style={halfWidth}
                >
                    {fieldTitle}
                </th>,
            );

            const fieldValue = messageHtmlToComponent(formatText(field.value, this.props.options));
            bodyCols.push(
                <td
                    className='attachment-field'
                    key={'attachment__field-' + i + '__' + nrTables}
                >
                    {fieldValue}
                </td>,
            );
            rowPos += 1;
            lastWasLong = !(field.short === true);
        });
        if (headerCols.length > 0) { // Flush last fields
            fieldTables.push(
                <table
                    className='attachment-fields'
                    key={'attachment__table__' + nrTables}
                >
                    <thead>
                        <tr>
                            {headerCols}
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            {bodyCols}
                        </tr>
                    </tbody>
                </table>,
            );
        }
        return (
            <div>
                {fieldTables}
            </div>
        );
    }
}
