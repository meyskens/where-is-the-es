package nmbs

const testResponse = `

<input type="hidden" id="trainSearchErrorResult" value="false" />

<div id="delayCertificateTrainRouteDetails" class="well theme-white marg-top-sm-20 custom-planner-route-detailed planner-delay">

    <ol>
        <li class="planner-dtl__item planner-head">
            <div>Aankomst</div>
            <div>Vertrek</div>
            <div>Tussenstops</div>
        </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                19:22
                            </div>
                                <div class="delay-txt">+ 1</div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot__icon-wrapper">

                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                <div class="train-origin-station" data-origin-station-time="19:22" data-origin-station-uic="8814001" data-origin-station-name="BRUSSEL-ZUID">BRUSSEL-ZUID</div>
                            </div>
                            <div id="trip-message-text" class="planner-dtl__detail">
                                INT 453 richting PRAHA HL.N.
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                19:58
                            </div>
                                <div class="delay-txt">+ 4</div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                20:02
                            </div>
                                <div class="delay-txt">+ 5</div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>ANTWERPEN-CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                20:41
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                20:44
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>ROOSENDAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                21:19
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                21:22
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>ROTTERDAM CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                21:40
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                21:42
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DEN HAAG HOLLANDS SPOOR</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                22:28
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                22:34
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>AMSTERDAM CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                23:08
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                23:13
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>AMERSFOORT CENTRAAL</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                23:48
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                23:52
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DEVENTER</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                06:16
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                06:20
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>BERLIN HBF</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                06:27
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                06:29
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>BERLIN OSTBAHNHOF</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                08:50
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                08:54
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DRESDEN HBF</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                09:21
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                09:23
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>BAD SCHANDAU</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item   is-partially-past">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                09:41
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                            <div>
                                <span class="sr-only">Vertrek in</span>
                                09:46
                            </div>
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div>DECIN HL.N.</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>
                <li class="planner-dtl__item planner-dtl__item--transfer  ">
                    <div class="planner-dtl__arrival">
                            <div>
                                <span class="sr-only">Aangekomen in</span>
                                11:24
                            </div>
                    </div>
                    <div class="planner-dtl__departure">
                    </div>
                        <div class="planner-dtl__timeline">
                            <div class="timeline__dot">
                                <div class="timeline__dot timeline__dot-transfer">
                                    <span class="timeline__dot-transfer-content shade"></span>
                                </div>
                            </div>
                        </div>
                        <div class="planner-dtl__content">
                            <div class="planner-dtl__lbl mobile-full-width">
                                <span class="sr-only">Van</span>
                                    <div class="train-destination-station" data-destination-station-time="11:24" data-destination-station-uic="5457076" data-destination-station-name="PRAHA HL.N.">PRAHA HL.N.</div>
                            </div>
                            <div class="js-dropdown dropdown planner-dtl__dropdown dropdown--open">
                                <ul class="dropdown__list"></ul>
                            </div>
                        </div>
                </li>

    </ol>
</div>`
